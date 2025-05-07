package keys

import (
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Config contains the dependencies needed to create a new key service.
type Config struct {
	// Logger for service operations
	Logger         logging.Logger
	// Database connection
	DB             db.Database
	// Clock for time-related operations
	Clock          clock.Clock
	// Cache for API keys by their hash
	KeyCache       cache.Cache[string, db.Key]
	// Cache for workspaces by their ID
	WorkspaceCache cache.Cache[string, db.Workspace]
}

type service struct {
	logger logging.Logger
	db     db.Database
	// hash -> key
	keyCache       cache.Cache[string, db.Key]
	workspaceCache cache.Cache[string, db.Workspace]
}

// New creates a new key service with the provided configuration.
// It returns a service that implements the KeyService interface.
//
// Example:
//
//	keySvc, err := keys.New(keys.Config{
//		Logger:         logger,
//		DB:             database,
//		Clock:          clock.New(),
//		KeyCache:       caches.KeyByHash,
//		WorkspaceCache: caches.WorkspaceByID,
//	})
//	if err != nil {
//		log.Fatalf("Failed to create key service: %v", err)
//	}
func New(config Config) (*service, error) {
	return &service{
		logger:         config.Logger,
		db:             config.DB,
		keyCache:       config.KeyCache,
		workspaceCache: config.WorkspaceCache,
	}, nil
}
