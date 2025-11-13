package auth

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/go/apps/gw/server"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type authenticator struct {
	logger logging.Logger
	keys   keys.KeyService
}

var _ Authenticator = (*authenticator)(nil)

// New creates a new authenticator with the given configuration.
func New(config Config) (Authenticator, error) {
	if err := assert.All(
		assert.NotNilAndNotZero(config.Logger, "Logger is required"),
		assert.NotNilAndNotZero(config.Keys, "Keys DB is required"),
	); err != nil {
		return nil, err
	}

	return &authenticator{
		logger: config.Logger,
		keys:   config.Keys,
	}, nil
}

// Authenticate processes API key authentication for the request.
func (a *authenticator) Authenticate(ctx context.Context, sess *server.Session, config *partitionv1.GatewayConfig) error {
	// Skip authentication if not configured or not enabled
	if config.GetAuthConfig() == nil {
		return nil
	}

	// Extract API key from Authorization header
	apiKey, err := a.extractAPIKey(sess)
	if err != nil {
		return err
	}

	// Verify the API key
	return a.verifyAPIKey(ctx, sess, apiKey, config)
}

// extractAPIKey extracts and validates the API key from the Authorization header.
func (a *authenticator) extractAPIKey(sess *server.Session) (string, error) {
	req := sess.Request()

	// Check for Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fault.New("missing authorization header",
			fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
			fault.Public("Authorization header required"),
		)
	}

	// Validate Bearer token format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fault.New("invalid authorization header format",
			fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
			fault.Public("Invalid authorization header format"),
		)
	}

	// Extract API key
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if apiKey == "" {
		return "", fault.New("empty api key",
			fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
			fault.Public("API key is required"),
		)
	}

	return apiKey, nil
}

// verifyAPIKey retrieves and validates the API key against the configured keyspace.
func (a *authenticator) verifyAPIKey(ctx context.Context, sess *server.Session, apiKey string, config *partitionv1.GatewayConfig) error {
	z := zen.Session{
		WorkspaceID: sess.WorkspaceID,
	}

	key, emit, err := a.keys.Get(ctx, &z, hash.Sha256(apiKey))
	defer emit()
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.KeyVerificationFailed.URN()),
			fault.Internal("failed to retrieve api key"),
			fault.Public("Internal server error"),
		)
	}

	// Validate keyspace - ensure key belongs to the correct keyspace
	if key.Key.KeyAuthID != config.GetAuthConfig().GetKeyAuthId() {
		a.logger.Warn("key belongs to different keyspace",
			"requestId", sess.RequestID(),
			"key_id", key.Key.ID,
			"expected_keyspace", config.GetAuthConfig().GetKeyAuthId(),
			"actual_keyspace", key.Key.KeyAuthID,
		)

		return fault.New("key belongs to different keyspace",
			fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
			fault.Public("Invalid API key"),
		)
	}

	// Check if API is deleted
	if key.Key.ApiDeletedAtM.Valid {
		return fault.New("api key belongs to deleted api",
			fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
			fault.Public("Invalid API key"),
		)
	}

	// Verify the key with verification options
	opts := []keys.VerifyOption{
		keys.WithCredits(1),      // Default cost
		keys.WithRateLimits(nil), // Auto-applied rate limits only
		keys.WithIPWhitelist(),
		// keys.WithTags([]string)
	}

	err = key.Verify(ctx, opts...)
	if err != nil {
		a.logger.Error("key verification failed",
			"requestId", sess.RequestID(),
			"key_id", key.Key.ID,
			"error", err.Error(),
		)

		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.KeyVerificationFailed.URN()),
			fault.Public("Internal server error"),
		)
	}

	// Check if key is valid after verification
	if key.Status != keys.StatusValid {
		switch key.Status {
		case keys.StatusRateLimited:
			return fault.New("api key rate limited",
				fault.Code(codes.Gateway.Auth.RateLimited.URN()),
				fault.Public("Rate limit exceeded"),
			)
		case keys.StatusValid, keys.StatusNotFound, keys.StatusDisabled,
			keys.StatusExpired, keys.StatusForbidden, keys.StatusInsufficientPermissions,
			keys.StatusUsageExceeded, keys.StatusWorkspaceDisabled,
			keys.StatusWorkspaceNotFound:
			fallthrough
		default:
			return fault.New("api key verification failed",
				fault.Code(codes.Gateway.Auth.Unauthorized.URN()),
				fault.Public("Invalid API key"),
			)
		}
	}

	return nil
}
