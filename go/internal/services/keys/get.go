package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func (s *service) GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, error) {
	ctx, span := tracing.Start(ctx, "keys.GetRootKey")
	defer span.End()

	rootKey, err := zen.Bearer(sess)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Internal("no bearer"),
			fault.Public("You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'."),
		)
	}

	key, err := s.Get(ctx, sess, rootKey)
	if err != nil {
		return nil, err
	}

	// For root keys, convert validation failures to proper fault errors immediately
	if !key.Valid {
		return nil, fault.Wrap(
			key.ToFault(),
			fault.Internal("invalid root key"),
			fault.Public("The provided root key is invalid."),
		)
	}

	return key, nil
}

func (s *service) Get(ctx context.Context, sess *zen.Session, rawKey string) (*KeyVerifier, error) {
	ctx, span := tracing.Start(ctx, "keys.Get")
	defer span.End()

	err := assert.NotEmpty(rawKey)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("rawKey is empty"))
	}

	h := hash.Sha256(rawKey)
	key, err := s.keyCache.SWR(ctx, h, func(ctx context.Context) (db.FindKeyForVerificationRow, error) {
		return db.Query.FindKeyForVerification(ctx, s.db.RO(), h)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return &KeyVerifier{
				Valid:   false,
				Status:  StatusNotFound,
				message: "key does not exist",
			}, nil
		}

		return nil, fault.Wrap(
			err,
			fault.Internal("unable to load key"),
			fault.Public("We could not load the requested key."),
		)
	}

	// ForWorkspace set but that doesn't exist
	if key.ForWorkspaceID.Valid && !key.ForWorkspaceEnabled.Valid {
		return &KeyVerifier{
			Valid:   false,
			Status:  StatusWorkspaceNotFound,
			message: "workspace not found",
		}, nil
	}

	if !key.WorkspaceEnabled || (key.ForWorkspaceEnabled.Valid && !key.ForWorkspaceEnabled.Bool) {
		return &KeyVerifier{
			Valid:   false,
			Status:  StatusWorkspaceDisabled,
			message: "workspace is disabled",
		}, nil
	}

	// The DB returns this in array format and an empty array if not found
	var roles, permissions []string
	var ratelimitArr [][]string
	var ratelimits []db.KeyFindForVerificationRatelimit
	json.Unmarshal(key.Roles.([]byte), &roles)
	json.Unmarshal(key.Permissions.([]byte), &permissions)
	json.Unmarshal(key.Ratelimits.([]byte), &ratelimitArr)

	authorizedWorkspaceID := key.WorkspaceID
	if key.ForWorkspaceID.Valid {
		authorizedWorkspaceID = key.ForWorkspaceID.String
	}

	sess.WorkspaceID = authorizedWorkspaceID
	kv := &KeyVerifier{
		Key:                   key,
		clickhouse:            s.clickhouse,
		rateLimiter:           s.raterLimiter,
		usageLimiter:          s.usageLimiter,
		AuthorizedWorkspaceID: authorizedWorkspaceID,
		rBAC:                  s.rbac,
		session:               sess,
		logger:                s.logger,
		message:               "",
		IsRootKey:             key.ForWorkspaceID.Valid,
		// By default we assume the key is valid unless proven otherwise
		Valid:       true,
		Status:      StatusValid,
		Ratelimits:  ratelimits,
		Roles:       roles,
		Permissions: permissions,
	}

	if key.DeletedAtM.Valid {
		kv.Status = StatusNotFound
		kv.Valid = false
		kv.message = "key is deleted"
		return kv, nil
	}

	if !key.Enabled {
		kv.Status = StatusDisabled
		kv.Valid = false
		kv.message = "the key is disabled"
		return kv, nil
	}

	if key.Expires.Valid && time.Now().After(key.Expires.Time) {
		kv.Status = StatusExpired
		kv.Valid = false
		kv.message = fmt.Sprintf("the key has expired on %s", key.Expires.Time.Format(time.RFC3339))
	}

	return kv, nil
}
