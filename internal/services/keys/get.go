package keys

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen"
)

// GetRootKey retrieves and validates a root key from the session's Authorization header.
// Root keys are special administrative keys that can access workspace-level operations.
// Validation failures are immediately converted to fault errors for root keys.
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

	key, err := s.getRootKey(ctx, sess, hash.Sha256(rootKey))
	if err != nil {
		return nil, err
	}

	if key.Status != StatusValid {
		return nil, fault.Wrap(
			key.ToFault(),
			fault.Internal("invalid root key"),
			fault.Public("The provided root key is invalid."),
		)
	}

	// A root key MUST have ForWorkspaceID set - this distinguishes it from a regular API key.
	// Without this check, a regular key could be used in the Authorization header and
	// gain access to root key operations using its own workspace as the target.
	// We return the same error as a non-existent key to avoid leaking that the key exists.
	if !key.Key.ForWorkspaceID.Valid {
		return nil, fault.New("not a root key",
			fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.Internal("key does not have ForWorkspaceID set - not a root key"),
			fault.Public("The provided root key is invalid."),
		)
	}

	key.AuthorizedWorkspaceID = key.Key.ForWorkspaceID.String

	logger.Set(ctx, slog.Group("auth",
		slog.String("workspace_id", key.AuthorizedWorkspaceID),
		slog.String("root_key_id", key.Key.ID),
	))

	return key, nil
}

// Get retrieves a key from the database and performs basic validation checks.
// It returns a KeyVerifier that can be used for further validation with specific options.
// For normal keys, validation failures are indicated by KeyVerifier.Valid=false.
func (s *service) Get(ctx context.Context, sess *zen.Session, sha256Hash string) (*KeyVerifier, error) {
	return s.get(ctx, sess, sha256Hash)
}

// get loads an API key through the regular verification cache.
func (s *service) get(ctx context.Context, sess *zen.Session, sha256Hash string) (*KeyVerifier, error) {
	ctx, span := tracing.Start(ctx, "keys.Get")
	defer span.End()

	startTime := time.Now()

	err := assert.NotEmpty(sha256Hash)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("sha256Hash is empty"))
	}

	key, hit, err := s.keyCache.SWR(ctx, sha256Hash, func(ctx context.Context) (keysdb.CachedKeyData, error) {
		// Use database retry with exponential backoff, skipping non-transient errors
		row, err := mysql.WithRetryContext(ctx, func() (keysdb.FindKeyForVerificationRow, error) {
			return keysdb.Query.FindKeyForVerification(ctx, s.db.RO(), sha256Hash)
		})
		if err != nil {
			return keysdb.CachedKeyData{}, err
		}

		// Parse IP whitelist once during cache population for performance
		parsedIPWhitelist := make(map[string]struct{})
		if row.IpWhitelist.Valid && row.IpWhitelist.String != "" {
			ips := strings.Split(row.IpWhitelist.String, ",")
			for _, ip := range ips {
				trimmed := strings.TrimSpace(ip)
				if trimmed != "" {
					parsedIPWhitelist[trimmed] = struct{}{}
				}
			}
		}

		// Decode roles / permissions / ratelimits once during cache population so
		// that cache hits don't re-parse these JSON columns on every verify call.
		roles, err := db.UnmarshalNullableJSONTo[[]string](row.Roles)
		if err != nil {
			return keysdb.CachedKeyData{}, fault.Wrap(err, fault.Internal("failed to unmarshal roles"))
		}
		if roles == nil {
			roles = []string{}
		}

		permissions, err := db.UnmarshalNullableJSONTo[[]string](row.Permissions)
		if err != nil {
			return keysdb.CachedKeyData{}, fault.Wrap(err, fault.Internal("failed to unmarshal permissions"))
		}
		if permissions == nil {
			permissions = []string{}
		}

		ratelimitArr, err := db.UnmarshalNullableJSONTo[[]keysdb.KeyFindForVerificationRatelimit](row.Ratelimits)
		if err != nil {
			return keysdb.CachedKeyData{}, fault.Wrap(err, fault.Internal("failed to unmarshal ratelimits"))
		}

		// Convert rate limits array to map (key name -> config).
		// Key rate limits take precedence over identity rate limits.
		ratelimitConfigs := make(map[string]keysdb.KeyFindForVerificationRatelimit, len(ratelimitArr))
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

		return keysdb.CachedKeyData{
			FindKeyForVerificationRow: row,
			ParsedIPWhitelist:         parsedIPWhitelist,
			Roles:                     roles,
			Permissions:               permissions,
			RatelimitConfigs:          ratelimitConfigs,
		}, nil
	}, caches.DefaultFindFirstOp)

	return s.newKeyVerifier(sess, key, hit, err, startTime)
}

// getRootKey loads a root key through the dedicated root-key cache.
func (s *service) getRootKey(ctx context.Context, sess *zen.Session, sha256Hash string) (*KeyVerifier, error) {
	ctx, span := tracing.Start(ctx, "keys.GetRootKeyByHash")
	defer span.End()

	startTime := time.Now()
	if err := assert.NotEmpty(sha256Hash); err != nil {
		return nil, fault.Wrap(err, fault.Internal("sha256Hash is empty"))
	}

	rootKey, hit, err := s.rootKeyCache.SWR(ctx, sha256Hash, func(ctx context.Context) (keysdb.CachedRootKeyData, error) {
		return s.loadRootKey(ctx, sha256Hash)
	}, caches.DefaultFindFirstOp)

	return s.newKeyVerifier(sess, asCachedKey(rootKey), hit, err, startTime)
}

// loadRootKey checks the new store first and falls back to the legacy store only
// when the hash is absent.
func (s *service) loadRootKey(ctx context.Context, sha256Hash string) (keysdb.CachedRootKeyData, error) {
	return mysql.WithRetryContext(ctx, func() (keysdb.CachedRootKeyData, error) {
		row, err := keysdb.Query.FindUnkeyRootKeyForAuthentication(ctx, s.db.RO(), sha256Hash)
		if err == nil {
			if row.DeletedAt.Valid {
				return keysdb.CachedRootKeyData{}, sql.ErrNoRows
			}
			return cacheUnkeyRootKey(row)
		}
		if !mysql.IsNotFound(err) {
			return keysdb.CachedRootKeyData{}, err
		}

		legacy, err := keysdb.Query.FindLegacyRootKeyForAuthentication(ctx, s.db.RO(), sha256Hash)
		if err != nil {
			return keysdb.CachedRootKeyData{}, err
		}
		return cacheLegacyRootKey(legacy)
	})
}

// cacheUnkeyRootKey converts a new-store query result into cache data.
func cacheUnkeyRootKey(row keysdb.FindUnkeyRootKeyForAuthenticationRow) (keysdb.CachedRootKeyData, error) {
	permissions, err := unmarshalPermissions(row.Permissions)
	if err != nil {
		return keysdb.CachedRootKeyData{}, err
	}
	expires := sql.NullTime{}
	if row.Expires.Valid {
		expires = sql.NullTime{Time: time.UnixMilli(row.Expires.Int64), Valid: true}
	}
	return keysdb.CachedRootKeyData{
		ID:                  row.ID,
		KeyAuthID:           "",
		WorkspaceID:         row.WorkspaceID,
		ForWorkspaceID:      row.WorkspaceID,
		Name:                row.Name,
		Expires:             expires,
		Enabled:             row.Enabled,
		ApiDeletedAtM:       sql.NullInt64{},
		WorkspaceEnabled:    true,
		ForWorkspaceEnabled: row.WorkspaceEnabled,
		Permissions:         permissions,
	}, nil
}

// cacheLegacyRootKey converts a legacy-store query result into cache data.
func cacheLegacyRootKey(row keysdb.FindLegacyRootKeyForAuthenticationRow) (keysdb.CachedRootKeyData, error) {
	permissions, err := unmarshalPermissions(row.Permissions)
	if err != nil {
		return keysdb.CachedRootKeyData{}, err
	}
	return keysdb.CachedRootKeyData{
		ID:                  row.ID,
		KeyAuthID:           row.KeyAuthID,
		WorkspaceID:         row.WorkspaceID,
		ForWorkspaceID:      row.ForWorkspaceID.String,
		Name:                row.Name,
		Expires:             row.Expires,
		Enabled:             row.Enabled,
		ApiDeletedAtM:       row.ApiDeletedAtM,
		WorkspaceEnabled:    row.WorkspaceEnabled,
		ForWorkspaceEnabled: row.ForWorkspaceEnabled,
		Permissions:         permissions,
	}, nil
}

// asCachedKey adapts root-key data to the shared lifecycle validator.
func asCachedKey(rootKey keysdb.CachedRootKeyData) keysdb.CachedKeyData {
	//nolint:exhaustruct
	return keysdb.CachedKeyData{
		FindKeyForVerificationRow: keysdb.FindKeyForVerificationRow{
			ID:                  rootKey.ID,
			KeyAuthID:           rootKey.KeyAuthID,
			WorkspaceID:         rootKey.WorkspaceID,
			ForWorkspaceID:      sql.NullString{String: rootKey.ForWorkspaceID, Valid: true},
			Name:                rootKey.Name,
			Expires:             rootKey.Expires,
			Enabled:             rootKey.Enabled,
			ApiDeletedAtM:       rootKey.ApiDeletedAtM,
			WorkspaceEnabled:    rootKey.WorkspaceEnabled,
			ForWorkspaceEnabled: rootKey.ForWorkspaceEnabled,
		},
		ParsedIPWhitelist: map[string]struct{}{},
		Roles:             []string{},
		Permissions:       rootKey.Permissions,
		RatelimitConfigs:  map[string]keysdb.KeyFindForVerificationRatelimit{},
	}
}

// unmarshalPermissions decodes a query's permission JSON into a non-nil slice.
func unmarshalPermissions(value any) ([]string, error) {
	permissions, err := db.UnmarshalNullableJSONTo[[]string](value)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to unmarshal permissions"))
	}
	if permissions == nil {
		return []string{}, nil
	}
	return permissions, nil
}

// newKeyVerifier applies lifecycle checks shared by API and root keys.
func (s *service) newKeyVerifier(sess *zen.Session, key keysdb.CachedKeyData, hit cache.CacheHit, loadErr error, startTime time.Time) (kv *KeyVerifier, err error) {
	defer func() {
		if kv == nil {
			return
		}
		// A key that passed Get is not decided yet: KeyVerifier.Verify records
		// its terminal status. newKeyVerifier records everything else, including
		// every root key, which never runs through Verify.
		if kv.Status == StatusValid && !kv.isRootKey {
			return
		}
		kv.recordStatus(kv.Status)
	}()

	err = loadErr
	if err != nil {
		if mysql.IsNotFound(err) {
			// nolint:exhaustruct
			return &KeyVerifier{
				Status:    StatusNotFound,
				message:   "key does not exist",
				session:   sess,
				region:    s.region,
				source:    s.source,
				startTime: startTime,
			}, nil
		}

		return nil, fault.Wrap(
			err,
			fault.Internal("unable to load key"),
			fault.Public("We could not load the requested key."),
		)
	}

	if hit == cache.Null {
		// nolint:exhaustruct
		return &KeyVerifier{
			Status:    StatusNotFound,
			message:   "key does not exist",
			session:   sess,
			region:    s.region,
			source:    s.source,
			startTime: startTime,
		}, nil
	}

	// ForWorkspace set but that doesn't exist
	if key.ForWorkspaceID.Valid && !key.ForWorkspaceEnabled.Valid {
		// nolint:exhaustruct
		return &KeyVerifier{
			Status:    StatusWorkspaceNotFound,
			message:   "workspace not found",
			session:   sess,
			region:    s.region,
			source:    s.source,
			startTime: startTime,
		}, nil
	}

	// Workspace is disabled or the key is not allowed to be used for workspace operations
	if !key.WorkspaceEnabled || (key.ForWorkspaceEnabled.Valid && !key.ForWorkspaceEnabled.Bool) {
		// nolint:exhaustruct
		kv = &KeyVerifier{
			Status:                StatusWorkspaceDisabled,
			message:               "workspace is disabled",
			session:               sess,
			rBAC:                  s.rbac,
			region:                s.region,
			source:                s.source,
			rateLimiter:           s.rateLimiter,
			usageLimiter:          s.usageLimiter,
			AuthorizedWorkspaceID: key.WorkspaceID,
			isRootKey:             key.ForWorkspaceID.Valid,
			Key:                   key.FindKeyForVerificationRow,
			startTime:             startTime,
			spentCredits:          0,
		}

		return kv, nil
	}

	kv = &KeyVerifier{
		tags:                  []string{},
		Key:                   key.FindKeyForVerificationRow,
		rateLimiter:           s.rateLimiter,
		usageLimiter:          s.usageLimiter,
		AuthorizedWorkspaceID: key.WorkspaceID,
		rBAC:                  s.rbac,
		session:               sess,
		region:                s.region,
		source:                s.source,
		message:               "",
		isRootKey:             key.ForWorkspaceID.Valid,
		startTime:             startTime,
		spentCredits:          0,

		// By default we assume the key is valid unless proven otherwise
		Status:            StatusValid,
		ratelimitConfigs:  key.RatelimitConfigs,
		parsedIPWhitelist: key.ParsedIPWhitelist, // Use pre-parsed IPs from cache
		Roles:             key.Roles,
		Permissions:       key.Permissions,
		RatelimitResults:  nil,
	}

	if key.DeletedAtM.Valid {
		kv.setInvalid(StatusNotFound, "key is deleted")
		return kv, nil
	}

	if key.ApiDeletedAtM.Valid {
		kv.setInvalid(StatusNotFound, "key is deleted")
		return kv, nil
	}

	if !key.Enabled {
		kv.setInvalid(StatusDisabled, "key is disabled")
		return kv, nil
	}

	if key.Expires.Valid && startTime.After(key.Expires.Time) {
		kv.setInvalid(StatusExpired, fmt.Sprintf("the key has expired on %s", key.Expires.Time.Format(time.RFC3339)))
		return kv, nil
	}

	return kv, nil
}
