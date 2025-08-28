package auth

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Authenticator defines the interface for request authentication.
type Authenticator interface {
	// Authenticate validates the request against the provided configuration.
	// Returns nil if authentication succeeds or is not required.
	Authenticate(ctx context.Context, sess *server.Session, config *partitionv1.GatewayConfig) error
}

// Config holds configuration for the authenticator.
type Config struct {
	// Logger for debugging and monitoring
	Logger logging.Logger

	// Keys service for API key validation
	Keys keys.KeyService
}
