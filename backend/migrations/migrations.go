// Package migrations implements a small, hand-rolled, versioned migration
// runner for schema changes that AutoMigrate cannot handle safely.
//
// AutoMigrate in main.go still runs and remains the safe path for additive
// schema changes (new tables/columns) because every past migration has been
// additive and there's no way to test a full rewrite against the live
// database from this environment. This package is the place to add real
// explicit migrations going forward for anything AutoMigrate can't do
// safely — renaming/dropping columns, backfills, non-null constraints on
// existing data, etc.
//
// Add new files here as sql/NNNN_description.sql, never edit an
// already-applied one. Files are applied in filename order, each inside
// its own transaction, and recorded in the schema_migrations tracking
// table so they only ever run once.
package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"gorm.io/gorm"

	"talkd-backend/logger"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

const sqlDir = "sql"

// bootstrapSQL guarantees the schema_migrations tracking table exists even
// before any embedded migration file has run, so we always have somewhere
// to check/record applied versions. It is intentionally identical in
// intent to sql/0001_create_schema_migrations.sql (which also runs, and is
// a no-op the second time thanks to IF NOT EXISTS).
const bootstrapSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// Migrate applies any embedded SQL migration files that have not yet been
// recorded as applied, in filename order, each inside its own transaction.
func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("migrations: failed to get underlying sql.DB: %w", err)
	}

	if _, err := sqlDB.Exec(bootstrapSQL); err != nil {
		return fmt.Errorf("migrations: failed to bootstrap schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(sqlFiles, sqlDir)
	if err != nil {
		return fmt.Errorf("migrations: failed to read embedded sql directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		version := name

		var count int
		row := sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version)
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("migrations: failed to check applied status of %q: %w", version, err)
		}
		if count > 0 {
			continue
		}

		content, err := sqlFiles.ReadFile(sqlDir + "/" + name)
		if err != nil {
			return fmt.Errorf("migrations: failed to read %q: %w", version, err)
		}

		tx, err := sqlDB.Begin()
		if err != nil {
			return fmt.Errorf("migrations: failed to begin transaction for %q: %w", version, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrations: failed to apply %q: %w", version, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
			version, time.Now(),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrations: failed to record %q as applied: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrations: failed to commit %q: %w", version, err)
		}

		logger.Info("migrations: applied", "version", version)
	}

	return nil
}
