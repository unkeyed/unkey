package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

//go:embed schema.sql
var schema string

// Database wraps the SQLite connection with VM-specific operations
type Database struct {
	db     *sql.DB
	tracer trace.Tracer
	logger *slog.Logger
}

// New creates a new database connection and ensures schema is up to date
func New(dataDir string) (*Database, error) {
	return NewWithLogger(dataDir, slog.Default())
}

// NewWithLogger creates a new database connection with a custom logger
func NewWithLogger(dataDir string, logger *slog.Logger) (*Database, error) {
	// Ensure data directory exists with secure permissions
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(dataDir, "metald.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for high-scale deployment
	db.SetMaxOpenConns(25)   // Limit concurrent connections
	db.SetMaxIdleConns(5)    // Maintain idle connections for reuse
	db.SetConnMaxLifetime(0) // Keep connections alive (SQLite benefit)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		tracer: otel.Tracer("metald/database"),
		logger: logger.With("component", "database"),
	}

	// Apply schema
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	database.logger.Info("database initialized successfully",
		slog.String("path", dbPath),
	)

	return database, nil
}

// migrate applies the database schema
func (d *Database) migrate() error {
	_, span := d.tracer.Start(context.Background(), "database.migrate")
	defer span.End()

	d.logger.Debug("applying database schema")

	_, err := d.db.Exec(schema)
	if err != nil {
		span.RecordError(err)
		d.logger.Error("failed to apply database schema",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	d.logger.Debug("database schema applied successfully")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// DB returns the underlying sql.DB for advanced operations
func (d *Database) DB() *sql.DB {
	return d.db
}
