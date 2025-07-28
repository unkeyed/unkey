package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// GetRootKey retrieves and validates a root key from the session's Authorization header.
// Root keys are special administrative keys that can access workspace-level operations.
// Validation failures are immediately converted to fault errors for root keys.
func (s *service) GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, func(), error) {
	ctx, span := tracing.Start(ctx, "keys.GetRootKey")
	defer span.End()

	rootKey, err := zen.Bearer(sess)
	if err != nil {
		return nil, emptyLog, fault.Wrap(err,
			fault.Internal("no bearer"),
			fault.Public("You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'."),
		)
	}

	key, log, err := s.Get(ctx, sess, rootKey)
	if key.Key.ForWorkspaceID.Valid {
		key.AuthorizedWorkspaceID = key.Key.ForWorkspaceID.String
	}
	sess.WorkspaceID = key.AuthorizedWorkspaceID
	if err != nil {
		return nil, log, err
	}

	if key.Status != StatusValid {
		return nil, log, fault.Wrap(
			key.ToFault(),
			fault.Internal("invalid root key"),
			fault.Public("The provided root key is invalid."),
		)
	}

	return key, log, nil
}

var emptyLog = func() {}

// Get retrieves a key from the database and performs basic validation checks.
// It returns a KeyVerifier that can be used for further validation with specific options.
// For normal keys, validation failures are indicated by KeyVerifier.Valid=false.
func (s *service) Get(ctx context.Context, sess *zen.Session, rawKey string) (*KeyVerifier, func(), error) {
	ctx, span := tracing.Start(ctx, "keys.Get")
	defer span.End()

	err := assert.NotEmpty(rawKey)
	if err != nil {
		return nil, emptyLog, fault.Wrap(err, fault.Internal("rawKey is empty"))
	}

	h := hash.Sha256(rawKey)
	key, hit, err := s.keyCache.SWR(ctx, h, func(ctx context.Context) (db.FindKeyForVerificationRow, error) {
		return db.Query.FindKeyForVerification(ctx, s.db.RO(), h)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			// nolint:exhaustruct
			return &KeyVerifier{
				Status:  StatusNotFound,
				message: "key does not exist",
			}, emptyLog, nil
		}

		return nil, emptyLog, fault.Wrap(
			err,
			fault.Internal("unable to load key"),
			fault.Public("We could not load the requested key."),
		)
	}

	if hit == cache.Null {
		// nolint:exhaustruct
		return &KeyVerifier{
			Status:  StatusNotFound,
			message: "key does not exist",
		}, nil, nil
	}

	// ForWorkspace set but that doesn't exist
	if key.ForWorkspaceID.Valid && !key.ForWorkspaceEnabled.Valid {
		// nolint:exhaustruct
		return &KeyVerifier{
			Status:  StatusWorkspaceNotFound,
			message: "workspace not found",
		}, emptyLog, nil
	}

	kv := &KeyVerifier{
		Status:  StatusWorkspaceDisabled,
		message: "workspace is disabled",
	}

	if !key.WorkspaceEnabled || (key.ForWorkspaceEnabled.Valid && !key.ForWorkspaceEnabled.Bool) {
		// nolint:exhaustruct
		return kv, kv.log, nil
	}

	// The DB returns this in array format and an empty array if not found
	var roles, permissions []string
	var ratelimitArr []db.KeyFindForVerificationRatelimit
	err = json.Unmarshal(key.Roles.([]byte), &roles)
	if err != nil {
		return nil, emptyLog, err
	}
	err = json.Unmarshal(key.Permissions.([]byte), &permissions)
	if err != nil {
		return nil, emptyLog, err
	}
	err = json.Unmarshal(key.Ratelimits.([]byte), &ratelimitArr)
	if err != nil {
		return nil, emptyLog, err
	}

	// Convert rate limits array to map (key name -> config)
	// Key rate limits take precedence over identity rate limits
	ratelimitConfigs := make(map[string]db.KeyFindForVerificationRatelimit)
	for _, rl := range ratelimitArr {
		existing, exists := ratelimitConfigs[rl.Name]
		if !exists {
			ratelimitConfigs[rl.Name] = rl
			continue
		}

		if rl.KeyID != "" && existing.IdentityID != "" {
			ratelimitConfigs[rl.Name] = rl
		}
	}

	kv = &KeyVerifier{
		Key:                   key,
		clickhouse:            s.clickhouse,
		rateLimiter:           s.raterLimiter,
		usageLimiter:          s.usageLimiter,
		AuthorizedWorkspaceID: key.WorkspaceID,
		rBAC:                  s.rbac,
		session:               sess,
		logger:                s.logger,
		message:               "",
		isRootKey:             key.ForWorkspaceID.Valid,

		// By default we assume the key is valid unless proven otherwise
		Status:           StatusValid,
		ratelimitConfigs: ratelimitConfigs,
		Roles:            roles,
		Permissions:      permissions,
		RatelimitResults: nil,
	}

	if key.DeletedAtM.Valid {
		kv.setInvalid(StatusNotFound, "key is deleted")
		return kv, kv.log, nil
	}

	if key.ApiDeletedAtM.Valid {
		kv.setInvalid(StatusNotFound, "key is deleted")
		return kv, kv.log, nil
	}

	if !key.Enabled {
		kv.setInvalid(StatusDisabled, "key is disabled")
		return kv, kv.log, nil
	}

	if key.Expires.Valid && time.Now().After(key.Expires.Time) {
		kv.setInvalid(StatusExpired, fmt.Sprintf("the key has expired on %s", key.Expires.Time.Format(time.RFC3339)))
		return kv, kv.log, nil
	}

	return kv, kv.log, nil
}
