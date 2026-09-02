package workos

import (
	"context"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// NewRoleMappingResolver adds WorkOS role mapping to a verified JWT resolver.
// The provider config selects this behavior without inferring it from the issuer.
func NewRoleMappingResolver(inner auth.Resolver) auth.Resolver {
	return resolverWithRoles{resolver: inner}
}

// resolverWithRoles maps verified WorkOS roles to API permissions.
type resolverWithRoles struct {
	resolver auth.Resolver
}

// Resolve delegates token verification and then maps roles for JWT principals.
func (r resolverWithRoles) Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	p, err := r.resolver.Resolve(ctx, sess)
	if err != nil || p == nil {
		return p, err
	}
	if p.Type != principal.TypeJWT {
		return p, nil
	}

	source, ok := p.Source.(principal.JWTSource)
	if err := assert.True(ok, "JWT principal must have a JWT source"); err != nil {
		return nil, err
	}

	p.Permissions = permissionsForRoles(p.WorkspaceID, source.Roles)
	return p, nil
}
