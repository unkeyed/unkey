package analytics

import (
	"context"
	"net/url"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/vault"
)

// connectionManager is the default implementation that manages per-workspace ClickHouse connections
type connectionManager struct {
	settingsCache   cache.Cache[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]
	connectionCache cache.Cache[string, clickhouse.ClickHouse]
	database        db.Database
	baseURL         string
	vault           *vault.Service
}

// ConnectionManagerConfig contains configuration for the connection manager
type ConnectionManagerConfig struct {
	SettingsCache cache.Cache[string, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow]
	Database      db.Database
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
		baseURL:         config.BaseURL,
		vault:           config.Vault,
	}, nil
}

// GetConnection returns a cached connection and settings for the workspace or creates a new one
func (m *connectionManager) GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	// Try to get cached connection
	conn, hit := m.connectionCache.Get(ctx, workspaceID)
	if hit == cache.Hit {
		// Still need to get settings
		settings, err := m.getSettings(ctx, workspaceID)
		if err != nil {
			return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, err
		}

		return conn, settings, nil
	}

	// Create new connection
	conn, settings, err := m.createConnection(ctx, workspaceID)
	if err != nil {
		return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, err
	}

	// Store in cache
	m.connectionCache.Set(ctx, workspaceID, conn)

	return conn, settings, nil
}

// getSettings retrieves the workspace settings from cache
func (m *connectionManager) getSettings(ctx context.Context, workspaceID string) (db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	settings, hit, err := m.settingsCache.SWR(ctx, workspaceID, func(ctx context.Context) (db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
		return db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(ctx, m.database.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.Wrap(err,
			fault.Public("Failed to fetch workspace analytics configuration"),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)
	}

	if hit == cache.Null || db.IsNotFound(err) {
		return db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.New(
			"workspace settings not found or null",
			fault.Public("ClickHouse analytics is not configured for this workspace"),
			fault.Code(codes.Data.Analytics.NotConfigured.URN()),
		)
	}

	return settings, nil
}

// createConnection creates a new ClickHouse connection for a workspace
func (m *connectionManager) createConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	settings, err := m.getSettings(ctx, workspaceID)
	if err != nil {
		return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, err
	}

	// Decrypt password using vault
	decrypted, err := m.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Encrypted: settings.ClickhouseWorkspaceSetting.PasswordEncrypted,
		Keyring:   settings.ClickhouseWorkspaceSetting.WorkspaceID,
	})
	if err != nil {
		return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.Wrap(err,
			fault.Public("Failed to connect to ClickHouse analytics database"),
			fault.Code(codes.Data.Analytics.ConnectionFailed.URN()),
		)
	}

	// Parse base URL and inject workspace-specific credentials
	parsedURL, err := url.Parse(m.baseURL)
	if err != nil {
		return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.Wrap(err,
			fault.Public("Invalid ClickHouse URL configuration"),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)
	}

	// Inject workspace credentials
	parsedURL.User = url.UserPassword(settings.ClickhouseWorkspaceSetting.Username, decrypted.GetPlaintext())
	conn, err := clickhouse.New(clickhouse.Config{
		URL: parsedURL.String(),
	})
	if err != nil {
		return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.Wrap(err,
			fault.Public("Failed to connect to ClickHouse analytics database"),
			fault.Code(codes.Data.Analytics.ConnectionFailed.URN()),
		)
	}

	return conn, settings, nil
}
