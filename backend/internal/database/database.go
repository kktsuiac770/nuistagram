package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func Init(databaseURL string, pool PoolConfig) (*sql.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(pool.MaxOpenConns)
	db.SetMaxIdleConns(pool.MaxIdleConns)
	db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	executableDir, err := os.Getwd()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	migrationsPath := fmt.Sprintf("file://%s/migrations", executableDir)
	log.Printf("Running database migrations from %s", migrationsPath)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)

	m.Up()

	return db, nil
}

func HealthCheck(db *sql.DB) error {
	if db == nil {
		return sql.ErrConnDone
	}
	return db.Ping()
}
