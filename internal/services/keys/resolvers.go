package keys

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/zen"
)

// RootKeyResolver matches bearers carrying the "unkey_" prefix, the format
// every minted root key uses. It delegates the lookup to the keys service
// (so caching, status handling, and audit emit are preserved) and converts
// the resulting *KeyVerifier into the auth.Principal handlers consume.
// Construct via NewRootKeyResolver and pass to auth.NewDispatcher.
type RootKeyResolver struct {
	keys KeyService
}

// NewRootKeyResolver builds an auth.Resolver that hands "unkey_"-prefixed
// bearers to the keys service for lookup and validation.
func NewRootKeyResolver(keys KeyService) *RootKeyResolver {
	return &RootKeyResolver{keys: keys}
}

// Try returns (nil, _, nil) when the bearer is missing or not
// "unkey_"-prefixed, so the dispatcher can fall through to the next
// resolver. When the prefix matches, it surfaces the lookup result or
// error so the dispatcher stops and reports it rather than silently
// moving on.
func (r *RootKeyResolver) Try(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	bearer, err := zen.Bearer(sess)
	if err != nil || !strings.HasPrefix(bearer, "unkey_") {
		return nil, nil, nil
	}
	kv, emit, err := r.keys.GetRootKey(ctx, sess)
	if err != nil {
		return nil, emit, err
	}
	return &auth.Principal{
		Scheme:      auth.SchemeRootKey,
		ID:          kv.Key.ID,
		DisplayName: kv.Key.Name.String,
		WorkspaceID: kv.AuthorizedWorkspaceID(),
		Permissions: kv.Permissions(),
	}, emit, nil
}
