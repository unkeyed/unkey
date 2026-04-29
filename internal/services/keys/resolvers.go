package keys

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/zen"
)

// RootKeyResolver is the catch-all in the auth chain: place it after the
// JWT resolver so JWTs are claimed first and anything else (root keys,
// garbage, malformed credentials) lands here. The keys service produces
// the established "valid root key" error messages on every failure mode
// (missing bearer, malformed header, key-not-found, disabled, etc.) so
// the public error stays consistent with what callers have always seen.
type RootKeyResolver struct {
	keys KeyService
}

// NewRootKeyResolver builds an auth.Resolver that hands any bearer to the
// keys service for lookup and validation.
func NewRootKeyResolver(keys KeyService) *RootKeyResolver {
	return &RootKeyResolver{keys: keys}
}

// Resolve always delegates to the keys service so every failure mode
// (missing/malformed Authorization header, unknown key, disabled key) is
// reported with the same root-key-flavored error message clients rely on.
// As the catch-all resolver this never returns (nil, nil, nil); it either
// produces a Principal or surfaces an error that terminates the chain.
func (r *RootKeyResolver) Resolve(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	kv, emit, err := r.keys.GetRootKey(ctx, sess)
	if err != nil {
		return nil, emit, err
	}
	return &auth.Principal{
		Scheme:      auth.SchemeRootKey,
		ID:          kv.Key.ID,
		DisplayName: kv.Key.Name.String,
		WorkspaceID: kv.AuthorizedWorkspaceID,
		Permissions: kv.Permissions,
		Authorizer:  kv,
	}, emit, nil
}
