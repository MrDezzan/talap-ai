package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Convert mysql://user:pass@host:port/dbname to MySQL Driver DSN: user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
func parseMySQLURL(url string) string {
	if !strings.HasPrefix(url, "mysql://") {
		return url
	}
	trimmed := strings.TrimPrefix(url, "mysql://")
	
	// find @
	parts := strings.SplitN(trimmed, "@", 2)
	if len(parts) != 2 {
		return url
	}
	userPass := parts[0]
	rest := parts[1]

	// find /
	restParts := strings.SplitN(rest, "/", 2)
	hostPort := restParts[0]
	dbAndQuery := ""
	if len(restParts) == 2 {
		dbAndQuery = restParts[1]
	}

	// split dbname and query
	dbParts := strings.SplitN(dbAndQuery, "?", 2)
	dbname := dbParts[0]
	
	// if hostPort doesn't have tcp(), wrap it
	if !strings.Contains(hostPort, "tcp(") {
		hostPort = fmt.Sprintf("tcp(%s)", hostPort)
	}

	return fmt.Sprintf("%s@%s/%s?charset=utf8mb4&parseTime=True&loc=Local", userPass, hostPort, dbname)
}

func InitDB() *gorm.DB {
	var dsn string
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		dsn = parseMySQLURL(databaseURL)
	} else {
		host   := getEnv("DB_HOST", "localhost")
		port   := getEnv("DB_PORT", "3306")
		user   := getEnv("DB_USER", "root")
		pass   := getEnv("DB_PASS", "")
		dbname := getEnv("DB_NAME", "talkd")

		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user, pass, host, port, dbname)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Connection pool tuning
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db
}
