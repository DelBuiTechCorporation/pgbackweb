package main

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/logger"
	"github.com/pressly/goose/v3"
)

func runMigrations(env config.Env) {
	logger.Info("checking database migrations...")

	// Configure goose
	goose.SetDialect("postgres")

	// Open database connection
	db, err := sql.Open("postgres", env.PBW_POSTGRES_CONN_STRING)
	if err != nil {
		logger.FatalError("error connecting to database for migrations", logger.KV{"error": err})
	}
	defer db.Close()

	// Get migrations directory path
	migrationDir := "./internal/database/migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		// Try absolute path for Docker environments
		wd, _ := os.Getwd()
		migrationDir = filepath.Join(wd, "internal/database/migrations")
	}

	// Run migrations
	if err := goose.Up(db, migrationDir); err != nil {
		logger.FatalError("error running database migrations", logger.KV{"error": err})
	}

	logger.Info("database migrations completed successfully")
}