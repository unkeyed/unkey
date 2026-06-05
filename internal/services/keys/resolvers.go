package keys

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// RootKeyResolver resolves bearer credentials with the existing root-key service.
type RootKeyResolver struct {
	keys KeyService
}

// NewRootKeyResolver creates a resolver that authenticates bearer root keys.
func NewRootKeyResolver(keys KeyService) *RootKeyResolver {
	return &RootKeyResolver{keys: keys}
}

// Resolve authenticates a root key.
func (r *RootKeyResolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	// Yield when no Authorization header is present so the chain surfaces a
	// generic missing-credentials error rather than a root-key-specific message.
	if sess == nil || sess.Request() == nil || sess.Request().Header.Get("Authorization") == "" {
		return nil, nil
	}

	// Ignore GetRootKey's verification emit callback here. Auth middleware only
	// establishes the caller principal; handlers emit action-specific audit events
	// after authorization.
	key, _, err := r.keys.GetRootKey(ctx, sess)
	if err != nil {
		return nil, err
	}

	name := key.Key.Name.String
	if name == "" {
		name = "root key"
	}

	var permissions []string
	if len(key.Permissions) > 0 {
		permissions = key.Permissions
	}

	return &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   key.Key.ID,
			Name: name,
			Type: principal.SubjectTypeRootKey,
		},
		Type: principal.TypeAPIKey,
		Source: principal.KeySource{
			KeyID:       key.Key.ID,
			KeySpaceID:  key.Key.KeyAuthID,
			Permissions: permissions,
		},
		WorkspaceID: key.AuthorizedWorkspaceID,
		Permissions: key.Permissions,
	}, nil
}
