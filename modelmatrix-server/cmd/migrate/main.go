package main

import (
	"flag"
	"fmt"
	"os"

	"modelmatrix-server/internal/infrastructure/db"
	"modelmatrix-server/migrations"
	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"
)

func main() {
	// Parse command line flags
	drop := flag.Bool("drop", false, "Drop all tables before migration")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, "stdout", ""); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting database migration")
	logger.Info("Environment: %s", cfg.Env)

	// Initialize database
	database, err := db.Init(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Drop tables if requested
	if *drop {
		logger.Warn("Dropping all tables...")
		if err := migrations.DropAll(database); err != nil {
			logger.Fatal("Failed to drop tables: %v", err)
		}
		logger.Info("All tables dropped")
	}

	// Run migrations
	logger.Info("Running migrations...")
	if err := migrations.Migrate(database); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}
	logger.Info("Migrations completed successfully")

	// Create additional indexes
	logger.Info("Creating indexes...")
	if err := migrations.CreateIndexes(database); err != nil {
		logger.Warn("Failed to create some indexes (may already exist): %v", err)
	}
	logger.Info("Indexes created")

	logger.Info("Database migration completed successfully")
}

