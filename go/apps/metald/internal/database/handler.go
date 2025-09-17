package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/unkeyed/unkey/go/apps/metald/migrations"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	_ "modernc.org/sqlite"
)

type Database struct {
	Queries Querier
	db      *sql.DB
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

	logger.Info("database opened", slog.String("path", dbPath))

	queries := New(db)
	database := &Database{
		Queries: queries,
		db:      db,
		tracer:  otel.Tracer("metald/database"),
		logger:  logger,
	}

	return database, nil
}

func (db *Database) Close() error {
	return db.db.Close()
}

func (db *Database) DB() *sql.DB {
	return db.db
}
