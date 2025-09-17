package registry

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/oklog/ulid/v2"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
	_ "modernc.org/sqlite"
)

// Registry manages asset metadata in SQLite
type Registry struct {
	db     *sql.DB
	logger *slog.Logger
}

// New creates a new asset registry
func New(dbPath string, logger *slog.Logger) (*Registry, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	r := &Registry{
		db:     db,
		logger: logger.With("component", "registry"),
	}

	// Initialize schema
	if err := r.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return r, nil
}

// Close closes the registry
func (r *Registry) Close() error {
	return r.db.Close()
}

// initSchema creates the database schema
func (r *Registry) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		status INTEGER NOT NULL,
		backend INTEGER NOT NULL,
		location TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		checksum TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		last_accessed_at INTEGER NOT NULL,
		reference_count INTEGER NOT NULL DEFAULT 0,
		build_id TEXT,
		source_image TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_assets_type ON assets(type);
	CREATE INDEX IF NOT EXISTS idx_assets_status ON assets(status);
	CREATE INDEX IF NOT EXISTS idx_assets_created_at ON assets(created_at);
	CREATE INDEX IF NOT EXISTS idx_assets_last_accessed_at ON assets(last_accessed_at);
	CREATE INDEX IF NOT EXISTS idx_assets_reference_count ON assets(reference_count);
	CREATE INDEX IF NOT EXISTS idx_assets_build_id ON assets(build_id);

	CREATE TABLE IF NOT EXISTS asset_labels (
		asset_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (asset_id, key),
		FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_asset_labels_key_value ON asset_labels(key, value);

	CREATE TABLE IF NOT EXISTS asset_leases (
		id TEXT PRIMARY KEY,
		asset_id TEXT NOT NULL,
		acquired_by TEXT NOT NULL,
		acquired_at INTEGER NOT NULL,
		expires_at INTEGER,
		FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_asset_leases_asset_id ON asset_leases(asset_id);
	CREATE INDEX IF NOT EXISTS idx_asset_leases_expires_at ON asset_leases(expires_at);
	`

	if _, err := r.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CreateAsset creates a new asset record
func (r *Registry) CreateAsset(asset *assetv1.Asset) error {
	// Generate ID if not provided
	if asset.GetId() == "" {
		asset.Id = ulid.Make().String()
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert asset
	query := `
		INSERT INTO assets (
			id, name, type, status, backend, location, size_bytes, checksum,
			created_by, created_at, last_accessed_at, reference_count,
			build_id, source_image
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(query,
		asset.GetId(), asset.GetName(), asset.GetType(), asset.GetStatus(), asset.GetBackend(),
		asset.GetLocation(), asset.GetSizeBytes(), asset.GetChecksum(),
		asset.GetCreatedBy(), asset.GetCreatedAt(), asset.GetLastAccessedAt(), asset.GetReferenceCount(),
		asset.GetBuildId(), asset.GetSourceImage(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert asset: %w", err)
	}

	// Insert labels
	for key, value := range asset.GetLabels() {
		_, err = tx.Exec(
			"INSERT INTO asset_labels (asset_id, key, value) VALUES (?, ?, ?)",
			asset.GetId(), key, value,
		)
		if err != nil {
			return fmt.Errorf("failed to insert label %s=%s: %w", key, value, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("created asset",
		slog.String("id", asset.GetId()),
		slog.String("name", asset.GetName()),
		slog.String("type", asset.GetType().String()),
	)

	return nil
}

// GetAsset retrieves an asset by ID
func (r *Registry) GetAsset(id string) (*assetv1.Asset, error) {
	//nolint:exhaustruct // Asset fields will be populated from database
	asset := &assetv1.Asset{
		Labels: make(map[string]string),
	}

	// Get asset
	query := `
		SELECT name, type, status, backend, location, size_bytes, checksum,
		       created_by, created_at, last_accessed_at, reference_count,
		       build_id, source_image
		FROM assets WHERE id = ?
	`

	err := r.db.QueryRow(query, id).Scan(
		&asset.Name, &asset.Type, &asset.Status, &asset.Backend,
		&asset.Location, &asset.SizeBytes, &asset.Checksum,
		&asset.CreatedBy, &asset.CreatedAt, &asset.LastAccessedAt, &asset.ReferenceCount,
		&asset.BuildId, &asset.SourceImage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("asset not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	asset.Id = id

	// Get labels
	rows, err := r.db.Query("SELECT key, value FROM asset_labels WHERE asset_id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get labels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}
		asset.Labels[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating labels: %w", err)
	}

	// Update last accessed time
	go r.updateLastAccessed(id)

	return asset, nil
}

// UpdateAsset updates an asset record
func (r *Registry) UpdateAsset(asset *assetv1.Asset) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update asset
	query := `
		UPDATE assets SET
			name = ?, type = ?, status = ?, backend = ?, location = ?,
			size_bytes = ?, checksum = ?, last_accessed_at = ?,
			reference_count = ?, build_id = ?, source_image = ?
		WHERE id = ?
	`

	_, err = tx.Exec(query,
		asset.GetName(), asset.GetType(), asset.GetStatus(), asset.GetBackend(), asset.GetLocation(),
		asset.GetSizeBytes(), asset.GetChecksum(), asset.GetLastAccessedAt(),
		asset.GetReferenceCount(), asset.GetBuildId(), asset.GetSourceImage(),
		asset.GetId(),
	)
	if err != nil {
		return fmt.Errorf("failed to update asset: %w", err)
	}

	// Update labels (delete and re-insert)
	if _, err := tx.Exec("DELETE FROM asset_labels WHERE asset_id = ?", asset.GetId()); err != nil {
		return fmt.Errorf("failed to delete labels: %w", err)
	}

	for key, value := range asset.GetLabels() {
		_, labelErr := tx.Exec(
			"INSERT INTO asset_labels (asset_id, key, value) VALUES (?, ?, ?)",
			asset.GetId(), key, value,
		)
		if labelErr != nil {
			return fmt.Errorf("failed to insert label %s=%s: %w", key, value, labelErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteAsset deletes an asset record
func (r *Registry) DeleteAsset(id string) error {
	// AIDEV-NOTE: CASCADE constraints handle cleanup of labels and leases
	_, err := r.db.Exec("DELETE FROM assets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	r.logger.Info("deleted asset", slog.String("id", id))
	return nil
}

// ListAssets lists assets with optional filters
func (r *Registry) ListAssets(filters ListFilters) ([]*assetv1.Asset, error) {
	query := "SELECT id FROM assets WHERE 1=1"
	args := []interface{}{}

	// Add filters
	if filters.Type != assetv1.AssetType_ASSET_TYPE_UNSPECIFIED {
		query += " AND type = ?"
		args = append(args, filters.Type)
	}

	if filters.Status != assetv1.AssetStatus_ASSET_STATUS_UNSPECIFIED {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	// Label filters require a subquery
	for key, value := range filters.Labels {
		query += " AND id IN (SELECT asset_id FROM asset_labels WHERE key = ? AND value = ?)"
		args = append(args, key, value)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	defer rows.Close()

	var assets []*assetv1.Asset
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan asset ID: %w", err)
		}

		asset, err := r.GetAsset(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get asset %s: %w", id, err)
		}

		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return assets, nil
}

// CreateLease creates a new asset lease
func (r *Registry) CreateLease(assetID, acquiredBy string, ttl time.Duration) (string, error) {
	leaseID := ulid.Make().String()
	acquiredAt := time.Now().Unix()

	var expiresAt *int64
	if ttl > 0 {
		exp := time.Now().Add(ttl).Unix()
		expiresAt = &exp
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert lease
	_, err = tx.Exec(
		"INSERT INTO asset_leases (id, asset_id, acquired_by, acquired_at, expires_at) VALUES (?, ?, ?, ?, ?)",
		leaseID, assetID, acquiredBy, acquiredAt, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create lease: %w", err)
	}

	// Increment reference count
	_, err = tx.Exec("UPDATE assets SET reference_count = reference_count + 1 WHERE id = ?", assetID)
	if err != nil {
		return "", fmt.Errorf("failed to increment reference count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("created lease",
		slog.String("lease_id", leaseID),
		slog.String("asset_id", assetID),
		slog.String("acquired_by", acquiredBy),
	)

	return leaseID, nil
}

// ReleaseLease releases an asset lease
func (r *Registry) ReleaseLease(leaseID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get asset ID from lease
	var assetID string
	err = tx.QueryRow("SELECT asset_id FROM asset_leases WHERE id = ?", leaseID).Scan(&assetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("lease not found: %s", leaseID)
		}
		return fmt.Errorf("failed to get lease: %w", err)
	}

	// Delete lease
	_, err = tx.Exec("DELETE FROM asset_leases WHERE id = ?", leaseID)
	if err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}

	// Decrement reference count
	_, err = tx.Exec("UPDATE assets SET reference_count = reference_count - 1 WHERE id = ?", assetID)
	if err != nil {
		return fmt.Errorf("failed to decrement reference count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("released lease",
		slog.String("lease_id", leaseID),
		slog.String("asset_id", assetID),
	)

	return nil
}

// GetExpiredLeases returns leases that have expired
func (r *Registry) GetExpiredLeases() ([]string, error) {
	query := "SELECT id FROM asset_leases WHERE expires_at IS NOT NULL AND expires_at < ?"

	rows, err := r.db.Query(query, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query expired leases: %w", err)
	}
	defer rows.Close()

	var leaseIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan lease ID: %w", err)
		}
		leaseIDs = append(leaseIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return leaseIDs, nil
}

// GetUnreferencedAssets returns assets with zero references
func (r *Registry) GetUnreferencedAssets(olderThan time.Duration) ([]*assetv1.Asset, error) {
	cutoff := time.Now().Add(-olderThan).Unix()

	query := `
		SELECT id FROM assets
		WHERE reference_count = 0
		AND last_accessed_at < ?
		ORDER BY last_accessed_at ASC
	`

	rows, err := r.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query unreferenced assets: %w", err)
	}
	defer rows.Close()

	var assets []*assetv1.Asset
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan asset ID: %w", err)
		}

		asset, err := r.GetAsset(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get asset %s: %w", id, err)
		}

		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return assets, nil
}

// updateLastAccessed updates the last accessed timestamp
func (r *Registry) updateLastAccessed(id string) {
	_, err := r.db.Exec(
		"UPDATE assets SET last_accessed_at = ? WHERE id = ?",
		time.Now().Unix(), id,
	)
	if err != nil {
		r.logger.Warn("failed to update last accessed time",
			slog.String("asset_id", id),
			slog.String("error", err.Error()),
		)
	}
}

// ListFilters defines filters for listing assets
type ListFilters struct {
	Type   assetv1.AssetType
	Status assetv1.AssetStatus
	Labels map[string]string
	Limit  int
	Offset int
}
