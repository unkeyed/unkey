package auth

import (
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Service authenticates requests into normalized principals.
type Service interface {
	// Authenticate resolves the request credential into a normalized principal.
	// Implementations try configured credential sources in deterministic order
	// and return the first principal whose resolver claims the request.
	Authenticate(ctx context.Context, sess *zen.Session) (*principal.Principal, error)
}

// New builds a service that tries each resolver in the order provided.
func New(resolvers ...Resolver) Service {
	return chain(resolvers)
}

type chain []Resolver

// Authenticate resolves credentials by asking each resolver to claim the request.
func (c chain) Authenticate(ctx context.Context, sess *zen.Session) (*principal.Principal, error) {
	// Reuse a principal that was already resolved earlier in the request.
	if p := sess.Principal(); p != nil {
		return p, nil
	}

	for _, resolver := range c {
		principal, err := resolver.Resolve(ctx, sess)
		if err != nil {
			return nil, err
		}
		if principal == nil {
			continue
		}

		sess.SetPrincipal(principal)
		logger.Set(ctx, slog.Group("auth",
			slog.String("type", string(principal.Type)),
			slog.String("workspace_id", principal.WorkspaceID),
			slog.String("subject", principal.Subject.ID),
			slog.String("subject_name", principal.Subject.Name),
			slog.Int("permission_count", len(principal.Permissions)),
		))

		return principal, nil
	}

	if _, err := zen.Bearer(sess); err != nil {
		return nil, err
	}

	return nil, fault.New("no credentials",
		fault.Code(codes.Auth.Authentication.Missing.URN()),
		fault.Internal("no resolver matched the request"),
		fault.Public("You must provide a bearer credential in the Authorization header."),
	)
}
