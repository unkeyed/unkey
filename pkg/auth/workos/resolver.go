package workos

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// NewPermissionTranslatingResolver decorates a JWT resolver so WorkOS
// permission slugs in the verified token are translated into canonical Unkey
// permissions before authorization. The wrapped resolver owns verification
// (issuer, audience, signature); this decorator only rewrites the permission
// set, so it composes with any issuer the operator configures for the entry.
func NewPermissionTranslatingResolver(inner auth.Resolver) auth.Resolver {
	return resolverWithPermissions{resolver: inner}
}

// resolverWithPermissions decorates a JWT resolver so WorkOS permission slugs
// are translated into canonical Unkey permissions before authorization.
type resolverWithPermissions struct {
	resolver auth.Resolver
}

// Resolve delegates to the wrapped resolver and rewrites the principal
// permission set; non-JWT principals pass through untouched.
func (r resolverWithPermissions) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	p, err := r.resolver.Resolve(ctx, sess)
	if err != nil || p == nil {
		return p, err
	}
	if p.Type != principal.TypeJWT {
		return p, nil
	}

	p.Permissions = translatePermissions(p.WorkspaceID, p.Permissions)
	return p, nil
}
