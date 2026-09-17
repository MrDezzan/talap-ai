-- NOTE: this project's database is MySQL (see backend/config/db.go), so this
-- uses MySQL-compatible types. MySQL cannot index a bare TEXT column as a
-- PRIMARY KEY, hence VARCHAR(255) rather than TEXT here.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
