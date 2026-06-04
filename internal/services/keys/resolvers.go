package keys

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth"
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

// Resolve authenticates a root key and intentionally ignores verification emit logging.
// The API action itself is still audited by handlers using the returned principal.
func (r *RootKeyResolver) Resolve(ctx context.Context, sess *zen.Session) (*auth.Principal, error) {
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

	return &auth.Principal{
		Version: auth.PrincipalVersion,
		Subject: auth.Subject{
			ID:   key.Key.ID,
			Name: name,
			Type: auth.SubjectTypeRootKey,
		},
		Type: auth.PrincipalTypeAPIKey,
		Source: auth.Source{
			Key: &auth.KeySource{
				KeyID:       key.Key.ID,
				KeySpaceID:  key.Key.KeyAuthID,
				Permissions: permissions,
			},
			JWT:           nil,
			PortalSession: nil,
		},
		WorkspaceID: key.AuthorizedWorkspaceID,
		Permissions: key.Permissions,
	}, nil
}
