package rootkey

import (
	"context"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver resolves bearer credentials with the existing root-key service.
type Resolver struct {
	keys keys.KeyService
}

// NewResolver creates a resolver that authenticates bearer root keys.
func NewResolver(keys keys.KeyService) *Resolver {
	return &Resolver{keys: keys}
}

// Resolve authenticates a root key.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
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
			Permissions: key.Permissions,
		},
		WorkspaceID: key.AuthorizedWorkspaceID,
		Permissions: key.Permissions,
	}, nil
}
