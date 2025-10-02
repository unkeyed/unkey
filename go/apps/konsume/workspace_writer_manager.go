package konsume

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/analytics"
	analyticsstorage "github.com/unkeyed/unkey/go/pkg/analytics/storage"
	icebergstorage "github.com/unkeyed/unkey/go/pkg/analytics/storage/iceberg"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// WorkspaceWriterManager manages per-workspace analytics writers with caching.
// It handles dynamic creation of dual writers that write to both Unkey's ClickHouse
// and customer-specific data lakes.
type WorkspaceWriterManager struct {
	// writers cache maps workspaceId -> analytics.Writer
	writerCache cache.Cache[string, analytics.Writer]

	// unkeyWriter is the primary writer that always receives events
	unkeyWriter analytics.Writer

	// database for loading workspace analytics configs
	db db.Database

	// vault for decrypting sensitive fields in config
	vault *vault.Service

	logger logging.Logger
}

// WorkspaceWriterManagerConfig contains configuration for the workspace writer manager
type WorkspaceWriterManagerConfig struct {
	// UnkeyWriter is the primary writer (always writes here)
	UnkeyWriter analytics.Writer

	// Database for loading workspace configs
	DB db.Database

	// Vault for decrypting sensitive config fields
	Vault *vault.Service

	// Clock for cache timing
	Clock clock.Clock

	Logger logging.Logger
}

// NewWorkspaceWriterManager creates a new workspace writer manager
func NewWorkspaceWriterManager(config WorkspaceWriterManagerConfig) (*WorkspaceWriterManager, error) {
	writerCache, err := cache.New(cache.Config[string, analytics.Writer]{
		Fresh:    time.Minute * 5,
		Stale:    time.Minute * 30,
		Logger:   config.Logger,
		MaxSize:  1_000_000,
		Resource: "workspace_analytics_writer",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create writer cache: %w", err)
	}

	return &WorkspaceWriterManager{
		writerCache: writerCache,
		unkeyWriter: config.UnkeyWriter,
		db:          config.DB,
		vault:       config.Vault,
		logger:      config.Logger,
	}, nil
}

// GetWriter returns a writer for the given workspace.
// If the workspace has a custom analytics config, it returns a dual writer
// that writes to both Unkey's ClickHouse and the workspace's custom storage.
// If no custom config exists, it returns just the Unkey ClickHouse writer.
func (m *WorkspaceWriterManager) GetWriter(ctx context.Context, workspaceID string) (analytics.Writer, error) {
	writer, hit, err := m.writerCache.SWR(ctx, workspaceID, func(ctx context.Context) (analytics.Writer, error) {
		return m.createWriter(ctx, workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		m.logger.Error("failed to create writer cache", "workspace_id", workspaceID, "error", err)
		return nil, err
	}

	// If cache hit is null or we got NotFound, no config exists - use unkey writer only
	if hit == cache.Null || db.IsNotFound(err) {
		return m.unkeyWriter, nil
	}

	return writer, nil
}

// createWriter creates a new writer for the workspace by loading config from DB
func (m *WorkspaceWriterManager) createWriter(ctx context.Context, workspaceID string) (analytics.Writer, error) {
	// Try to load workspace analytics config
	config, err := db.Query.FindAnalyticsConfigByWorkspaceID(ctx, m.db.RO(), workspaceID)
	if err != nil {
		return nil, err
	}

	// Check if analytics is enabled for this workspace
	if !config.Enabled {
		m.logger.Debug("analytics disabled for workspace",
			"workspace_id", workspaceID,
		)
		// Return not found error so cache stores as null
		return nil, sql.ErrNoRows
	}

	// Create workspace-specific writer based on storage type
	var workspaceWriter analytics.Writer
	switch config.Storage {
	case db.AnalyticsConfigStorageIceberg:
		// Parse config into typed struct
		var icebergConfig icebergstorage.WorkspaceConfig
		if err := json.Unmarshal(config.Config, &icebergConfig); err != nil {
			m.logger.Error("failed to parse iceberg config for workspace, treating as not configured",
				"workspace_id", workspaceID,
				"error", err.Error(),
			)

			return nil, sql.ErrNoRows
		}

		// Decrypt encrypted fields
		if err := icebergConfig.Decrypt(ctx, config.WorkspaceID, m.vault); err != nil {
			m.logger.Error("failed to decrypt iceberg config for workspace, treating as not configured",
				"workspace_id", workspaceID,
				"error", err.Error(),
			)
			return nil, sql.ErrNoRows
		}

		// Log decrypted credentials for debugging (REMOVE IN PRODUCTION!)
		m.logger.Warn("decrypted iceberg config for workspace",
			"workspace_id", workspaceID,
			"bucket", icebergConfig.Bucket,
			"endpoint", icebergConfig.Endpoint,
			"region", icebergConfig.Region,
			"access_key_id", icebergConfig.AccessKeyId,
			"secret_access_key", icebergConfig.SecretAccessKey,
		)

		workspaceWriter, err = m.createIcebergWriter(&icebergConfig)
		if err != nil {
			m.logger.Error("failed to create iceberg writer for workspace, treating as not configured",
				"workspace_id", workspaceID,
				"error", err.Error(),
			)
			return nil, sql.ErrNoRows
		}

		m.logger.Info("iceberg writer created for workspace", "workspace_id", workspaceID)

	case db.AnalyticsConfigStorageClickhouse:
		// Customer wants their own ClickHouse
		m.logger.Warn("clickhouse storage type not yet implemented for workspace configs",
			"workspace_id", workspaceID,
		)

		return nil, sql.ErrNoRows

	default:
		m.logger.Warn("unknown storage type for workspace analytics config",
			"workspace_id", workspaceID,
			"storage_type", config.Storage,
		)

		return nil, sql.ErrNoRows
	}

	// Create dual writer: Unkey (primary) + Workspace (secondary)
	dualConfig := analyticsstorage.DualConfig{
		FailOnPrimaryError:   true,  // Always fail if Unkey writer fails
		FailOnSecondaryError: false, // Don't fail if workspace writer fails
	}

	dualWriter := analyticsstorage.NewDualWriter(
		m.unkeyWriter,
		workspaceWriter,
		dualConfig,
		m.logger,
	)

	m.logger.Info("created dual writer for workspace",
		"workspace_id", workspaceID,
		"storage_type", config.Storage,
	)

	return dualWriter, nil
}

// createIcebergWriter creates an Iceberg writer from workspace config
func (m *WorkspaceWriterManager) createIcebergWriter(config *icebergstorage.WorkspaceConfig) (analytics.Writer, error) {
	// Validate required fields
	if config.Bucket == "" {
		return nil, fmt.Errorf("missing required field: bucket")
	}

	// Use default format if not specified
	format := config.Format
	if format == "" {
		format = "iceberg"
	}

	// Create iceberg config
	icebergConfig := icebergstorage.Config{
		Endpoint:        config.Endpoint,
		CatalogEndpoint: config.CatalogEndpoint,
		CatalogToken:    config.CatalogToken,
		Region:          config.Region,
		AccessKeyID:     config.AccessKeyId,
		SecretAccessKey: config.SecretAccessKey,
		Bucket:          config.Bucket, // Use full bucket name, not prefix
		Format:          format,
	}

	return icebergstorage.New(icebergConfig, m.logger)
}

// InvalidateCache removes a workspace from the cache, forcing it to be reloaded
// on next access. Useful when workspace analytics config is updated.
func (m *WorkspaceWriterManager) InvalidateCache(ctx context.Context, workspaceID string) {
	m.writerCache.Remove(ctx, workspaceID)
	m.logger.Info("invalidated cached writer for workspace", "workspace_id", workspaceID)
}

// Close closes all cached writers
func (m *WorkspaceWriterManager) Close(ctx context.Context) error {
	m.logger.Info("closing workspace writer manager")
	// Cache handles its own cleanup
	return nil
}
