package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Database struct {
	Queries Querier
	tracer  trace.Tracer
	logger  *slog.Logger
}

func NewDatabase(dataDir string) (*Database, error) {
	return NewDatabaseWithLogger(dataDir, slog.Default())
}

func NewDatabaseWithLogger(dataDir string, logger *slog.Logger) (*Database, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "metald.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := New(db)

	database := &Database{
		Queries: queries,
		tracer:  otel.Tracer("metald/database"),
		logger:  logger,
	}

	return database, nil
}
