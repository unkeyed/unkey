package keys

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// KeyVerifier represents a key that has been loaded from the database and is ready for verification.
// It contains all the necessary information and services to perform various validation checks.
type KeyVerifier struct {
	Key                   db.FindKeyForVerificationRow         // The key data from the database
	Ratelimits            []db.KeyFindForVerificationRatelimit // Rate limits configured for this key
	Roles                 []string                             // RBAC roles assigned to this key
	Permissions           []string                             // RBAC permissions assigned to this key
	Valid                 bool                                 // Whether the key passed all validation checks
	Status                KeyStatus                            // The current validation status
	AuthorizedWorkspaceID string                               // The workspace ID this key is authorized for
	RatelimitResponses    []ratelimit.RatelimitResponse        // Responses from rate limit checks
	isRootKey             bool                                 // Whether this is a root key (special handling)
	session               *zen.Session                         // The current request session
	rateLimiter           ratelimit.Service                    // Rate limiting service
	usageLimiter          usagelimiter.Service                 // Usage limiting service
	rBAC                  *rbac.RBAC                           // Role-based access control service
	clickhouse            clickhouse.ClickHouse                // Clickhouse for telemetry
	logger                logging.Logger                       // Logger for verification operations
	message               string                               // Internal message for validation failures
}

// Verify performs key verification with the given options.
// For root keys: returns fault errors for validation failures.
// For normal keys: returns error only for system problems, check k.Valid and k.Status for validation results.
func (k *KeyVerifier) Verify(ctx context.Context, opts ...VerifyOption) error {
	// Skip verification if key is already invalid
	if !k.Valid {
		// For root keys, auto-return validation failures as fault errors
		if k.isRootKey {
			return k.ToFault()
		}
		return nil
	}

	config := &verifyConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return err
		}
	}

	var err error
	if config.credits != nil {
		err = k.withCredits(ctx, *config.credits)
		if err != nil {
			return err
		}
	}

	if config.ipWhitelist {
		err = k.withIPWhitelist()
		if err != nil {
			return err
		}
	}

	if config.permissions != nil {
		err = k.withPermissions(ctx, *config.permissions)
		if err != nil {
			return err
		}
	}

	if len(config.ratelimits) > 0 {
		err = k.withRateLimits(ctx, config.ratelimits)
		if err != nil {
			return err
		}
	}

	// Do this somewhere else lol
	// Buffer telemetry data
	k.clickhouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
		RequestID:   k.session.RequestID(),
		WorkspaceID: k.session.AuthorizedWorkspaceID(),
		Time:        time.Now().UnixMilli(),
		Region:      "",
		Outcome:     string(k.Status),
		KeySpaceID:  k.Key.KeyAuthID,
		KeyID:       k.Key.ID,
		IdentityID:  k.Key.IdentityID.String,
		Tags:        []string{},
	})

	// For root keys, auto-return validation failures as fault errors
	if k.isRootKey && !k.Valid {
		return k.ToFault()
	}

	return nil
}
