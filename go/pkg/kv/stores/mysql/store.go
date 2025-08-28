package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/kv"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// Config defines the parameters needed to establish database connections.
type Config struct {
	// The primary DSN for your database. This must support both reads and writes.
	PrimaryDSN string

	// The readonly replica will be used for most read queries.
	// If omitted, the primary is used.
	ReadOnlyDSN string

	// Logger for database-related operations
	Logger logging.Logger
}

// Store implements the kv.Store interface using MySQL
type Store struct {
	primary  *sql.DB
	readonly *sql.DB
	queries  *Queries
	logger   logging.Logger
}

func open(dsn string, logger logging.Logger) (db *sql.DB, err error) {
	if !strings.Contains(dsn, "parseTime=true") {
		return nil, fault.New("DSN must contain parseTime=true")
	}

	err = retry.New(
		retry.Attempts(3),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n) * time.Second
		}),
	).Do(func() error {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			logger.Info("mysql not ready yet, retrying...", "error", err.Error())
		}
		return err
	})

	return db, err
}

// NewStore creates a new MySQL-backed KV store
func NewStore(config Config) (kv.Store, error) {
	primary, err := open(config.PrimaryDSN, config.Logger)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("cannot open primary database"))
	}

	readonly := primary // Default to primary for reads
	if config.ReadOnlyDSN != "" {
		readonly, err = open(config.ReadOnlyDSN, config.Logger)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("cannot open readonly database"))
		}
		config.Logger.Info("kv store configured with separate read replica")
	} else {
		config.Logger.Info("kv store configured without separate read replica, using primary for reads")
	}

	return &Store{
		primary:  primary,
		readonly: readonly,
		queries:  New(primary),
		logger:   config.Logger,
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, bool, error) {
	now := time.Now().UnixMilli()

	// Use readonly connection for Get operations
	queries := New(s.readonly)
	row, err := queries.Get(ctx, GetParams{
		Key: key,
		Ttl: sql.NullInt64{Int64: now, Valid: true},
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	// Check if TTL is expired and delete if so
	if row.Ttl.Valid && row.Ttl.Int64 <= now {
		// Delete the expired key using primary connection
		err = s.queries.DeleteExpired(ctx, DeleteExpiredParams{
			Key: key,
			Ttl: sql.NullInt64{Int64: now, Valid: true},
		})
		if err != nil {
			s.logger.Warn("failed to delete expired key", "key", key, "error", err.Error())
		}
		return nil, false, nil
	}

	return row.Value, true, nil
}

func (s *Store) Set(ctx context.Context, key string, workspaceID string, value []byte, ttl *time.Duration) error {
	now := time.Now().UnixMilli()

	var ttlValue sql.NullInt64
	if ttl != nil {
		ttlMs := now + ttl.Milliseconds()
		ttlValue = sql.NullInt64{Int64: ttlMs, Valid: true}
	}

	err := s.queries.Set(ctx, SetParams{
		Key:         key,
		WorkspaceID: workspaceID,
		Value:       value,
		Ttl:         ttlValue,
		CreatedAt:   now,
	})

	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.queries.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

func (s *Store) ListByWorkspace(ctx context.Context, workspaceID string, cursor int64, limit int) ([]kv.KvEntry, error) {
	now := time.Now().UnixMilli()

	// cursor = 0 means start from the beginning (oldest records first)

	// Use readonly connection for List operations
	queries := New(s.readonly)
	rows, err := queries.ListByWorkspace(ctx, ListByWorkspaceParams{
		WorkspaceID: workspaceID,
		Ttl:         sql.NullInt64{Int64: now, Valid: true},
		ID:          cursor,
		Limit:       int32(limit),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list by workspace: %w", err)
	}

	return s.convertRows(rows)
}

func (s *Store) convertRows(rows []Kv) ([]kv.KvEntry, error) {
	entries := make([]kv.KvEntry, len(rows))

	for i, row := range rows {
		var ttl *int64
		if row.Ttl.Valid {
			ttl = &row.Ttl.Int64
		}

		entries[i] = kv.KvEntry{
			ID:          row.ID,
			Key:         row.Key,
			WorkspaceID: row.WorkspaceID,
			Value:       row.Value,
			TTL:         ttl,
			CreatedAt:   row.CreatedAt,
		}
	}

	return entries, nil
}

// Close closes the database connections
func (s *Store) Close() error {
	var errs []error

	if err := s.primary.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close primary connection: %w", err))
	}

	if s.readonly != s.primary {
		if err := s.readonly.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close readonly connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}
