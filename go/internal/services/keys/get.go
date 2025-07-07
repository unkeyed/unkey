package keys

import (
	"context"
	"encoding/json"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func (s *service) Get(ctx context.Context, sess *zen.Session, rawKey string) (*KeyVerifier, error) {
	ctx, span := tracing.Start(ctx, "keys.GetRootKey")
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
				Valid:  false,
				Status: openapi.NOTFOUND,
			}, nil
		}

		return &KeyVerifier{}, fault.Wrap(
			err,
			fault.Internal("unable to load key"),
			fault.Public("We could not load the requested key."),
		)
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
		IsRootKey:             key.ForWorkspaceEnabled.Valid,
		AuthorizedWorkspaceID: authorizedWorkspaceID,
		rBAC:                  s.rbac,
		session:               sess,
		// By default we assume the key is valid unless proven otherwise
		Valid:       true,
		Status:      openapi.VALID,
		error:       nil,
		Ratelimits:  ratelimits,
		Roles:       roles,
		Permissions: permissions,
	}

	if key.DeletedAtM.Valid {
		kv.Status = openapi.NOTFOUND
		kv.Valid = false
		return kv, nil
	}

	if !key.Enabled {
		kv.Status = openapi.DISABLED
		kv.Valid = false
		return kv, nil
	}

	// if !key.WorkspaceEnabled || (key.ForWorkspaceEnabled.Valid && !key.ForWorkspaceEnabled.Bool) {
	// 	return VerifyResponse{}, fault.New(
	// 		"workspace is disabled",
	// 		fault.Code(codes.Auth.Authorization.WorkspaceDisabled.URN()),
	// 		fault.Internal("workspace disabled"),
	// 		fault.Public("The workspace is disabled."),
	// 	)
	// }

	if key.Expires.Valid && time.Now().After(key.Expires.Time) {
		kv.Status = openapi.EXPIRED
		kv.Valid = false
	}

	return kv, nil
}

func (s *service) GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, error) {
	rootKey, err := zen.Bearer(sess)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Internal("no bearer"),
			fault.Public("You must provide a valid root key in the Authorization header in the format 'Bearer ROOT_KEY'."),
		)
	}

	key, err := s.Get(ctx, sess, rootKey)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Internal("invalid root key"),
			fault.Public("The provided root key is invalid."))
	}

	sess.WorkspaceID = key.AuthorizedWorkspaceID

	return key, nil
}
