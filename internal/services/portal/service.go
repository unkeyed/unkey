package portal

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// SessionInfo contains the resolved identity from a portal browser session.
// It provides workspace and user scoping for existing API handlers.
type SessionInfo struct {
	WorkspaceID    string
	ExternalID     string
	PortalConfigID string
	Permissions    []string
	Metadata       map[string]any
	Preview        bool
}

// Service defines the interface for portal session operations.
type Service interface {
	// GetSession validates a portal session token and returns session info
	// for scoping existing handlers by workspace and external user identity.
	GetSession(ctx context.Context, token string) (*SessionInfo, error)
}

// Config holds the configuration for creating a new portal service instance.
type Config struct {
	DB           db.Database
	SessionCache cache.Cache[string, db.PortalSession]
}

type service struct {
	db           db.Database
	sessionCache cache.Cache[string, db.PortalSession]
}

// New creates a new portal service instance.
func New(config Config) Service {
	return &service{
		db:           config.DB,
		sessionCache: config.SessionCache,
	}
}
