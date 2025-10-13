package analytics

import (
	"context"
	"net/url"
	"time"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// ConnectionManager is the interface for managing per-workspace ClickHouse connections for analytics
type ConnectionManager interface {
	GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.ClickhouseWorkspaceSetting, error)
}

// connectionManager is the default implementation that manages per-workspace ClickHouse connections
type connectionManager struct {
	settingsCache   cache.Cache[string, db.ClickhouseWorkspaceSetting]
	connectionCache cache.Cache[string, clickhouse.ClickHouse]
	database        db.Database
	logger          logging.Logger
	baseURL         string
	vault           *vault.Service
}

// ConnectionManagerConfig contains configuration for the connection manager
type ConnectionManagerConfig struct {
	SettingsCache cache.Cache[string, db.ClickhouseWorkspaceSetting]
	Database      db.Database
	Logger        logging.Logger
	Clock         clock.Clock
	BaseURL       string // e.g., "http://clickhouse:8123/default" or "clickhouse://clickhouse:9000/default"
	Vault         *vault.Service
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config ConnectionManagerConfig) (ConnectionManager, error) {
	err := assert.All(
		assert.NotNilAndNotZero(config.Vault, "vault is required"),
		assert.NotNilAndNotZero(config.SettingsCache, "settings cache is required"),
		assert.NotNilAndNotZero(config.Database, "database is required"),
		assert.NotNilAndNotZero(config.Logger, "logger is required"),
		assert.NotNilAndNotZero(config.Clock, "clock is required"),
		assert.NotNilAndNotZero(config.BaseURL, "base URL is required"),
	)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Validation.AssertionFailed.URN()),
			fault.Public("Analytics are not configured for this instance"),
		)
	}

	// Create cache for ClickHouse connections
	connectionCache, err := cache.New(cache.Config[string, clickhouse.ClickHouse]{
		// It's fine to keep a long cache time for this.
		Fresh:    24 * time.Hour,
		Stale:    24 * time.Hour,
		Logger:   config.Logger,
		MaxSize:  1_000,
		Resource: "clickhouse_analytics_connection",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to create connection cache"))
	}

	return &connectionManager{
		settingsCache:   config.SettingsCache,
		connectionCache: connectionCache,
		database:        config.Database,
		logger:          config.Logger,
		baseURL:         config.BaseURL,
		vault:           config.Vault,
	}, nil
}

// GetConnection returns a cached connection and settings for the workspace or creates a new one
func (m *connectionManager) GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.ClickhouseWorkspaceSetting, error) {
	// Try to get cached connection
	conn, hit := m.connectionCache.Get(ctx, workspaceID)
	if hit == cache.Hit {
		// Still need to get settings
		settings, err := m.getSettings(ctx, workspaceID)
		if err != nil {
			return nil, db.ClickhouseWorkspaceSetting{}, err
		}

		return conn, settings, nil
	}

	// Create new connection
	conn, settings, err := m.createConnection(ctx, workspaceID)
	if err != nil {
		return nil, db.ClickhouseWorkspaceSetting{}, err
	}

	// Store in cache
	m.connectionCache.Set(ctx, workspaceID, conn)

	return conn, settings, nil
}

// getSettings retrieves the workspace settings from cache
func (m *connectionManager) getSettings(ctx context.Context, workspaceID string) (db.ClickhouseWorkspaceSetting, error) {
	settings, hit, err := m.settingsCache.SWR(ctx, workspaceID, func(ctx context.Context) (db.ClickhouseWorkspaceSetting, error) {
		return db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(ctx, m.database.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return db.ClickhouseWorkspaceSetting{}, fault.New(
				"workspace settings not found",
				fault.Public("ClickHouse analytics is not configured for this workspace"),
				fault.Code(codes.Data.Analytics.NotConfigured.URN()),
			)
		}

		return db.ClickhouseWorkspaceSetting{}, fault.Wrap(err,
			fault.Public("Failed to fetch workspace analytics configuration"),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)
	}

	if hit == cache.Null {
		return db.ClickhouseWorkspaceSetting{}, fault.New(
			"workspace settings null",
			fault.Public("ClickHouse analytics is not configured for this workspace"),
			fault.Code(codes.Data.Analytics.NotConfigured.URN()),
		)
	}

	return settings, nil
}

// createConnection creates a new ClickHouse connection for a workspace
func (m *connectionManager) createConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.ClickhouseWorkspaceSetting, error) {
	settings, err := m.getSettings(ctx, workspaceID)
	if err != nil {
		return nil, db.ClickhouseWorkspaceSetting{}, err
	}

	// Decrypt password using vault
	decrypted, err := m.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Encrypted: settings.PasswordEncrypted,
		Keyring:   settings.WorkspaceID,
	})
	if err != nil {
		return nil, db.ClickhouseWorkspaceSetting{}, fault.Wrap(err,
			fault.Public("Failed to decrypt workspace credentials"),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)
	}

	// Parse base URL and inject workspace-specific credentials
	parsedURL, err := url.Parse(m.baseURL)
	if err != nil {
		return nil, db.ClickhouseWorkspaceSetting{}, fault.Wrap(err,
			fault.Public("Invalid ClickHouse URL configuration"),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)
	}

	// Inject workspace credentials
	parsedURL.User = url.UserPassword(settings.Username, decrypted.GetPlaintext())
	conn, err := clickhouse.New(clickhouse.Config{
		URL:    parsedURL.String(),
		Logger: m.logger,
	})
	if err != nil {
		return nil, db.ClickhouseWorkspaceSetting{}, fault.Wrap(err,
			fault.Public("Failed to connect to ClickHouse analytics database"),
			fault.Code(codes.Data.Analytics.ConnectionFailed.URN()),
		)
	}

	return conn, settings, nil
}

// noopConnectionManager is a no-op implementation that returns errors indicating analytics is not configured
type noopConnectionManager struct{}

// NewNoopConnectionManager creates a new no-op connection manager for when analytics is not configured
func NewNoopConnectionManager() ConnectionManager {
	return &noopConnectionManager{}
}

// GetConnection always returns an error indicating analytics is not configured
func (m *noopConnectionManager) GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.ClickhouseWorkspaceSetting, error) {
	return nil, db.ClickhouseWorkspaceSetting{}, fault.New(
		"analytics not configured",
		fault.Code(codes.Data.Analytics.NotConfigured.URN()),
		fault.Public("Analytics are not configured for this instance"),
	)
}
