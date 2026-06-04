package auth

import (
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Service authenticates requests and authorizes the resolved principal.
type Service interface {
	// Authenticate resolves the request credential into a normalized principal.
	// Implementations try configured credential sources in deterministic order
	// and return the first principal whose resolver claims the request.
	Authenticate(ctx context.Context, sess *zen.Session) (*Principal, error)

	// Authorize evaluates the already-authenticated principal against an RBAC
	// query. A nil principal is treated as an authentication failure because
	// callers must authenticate before checking permissions.
	Authorize(ctx context.Context, principal *Principal, query rbac.PermissionQuery) error
}

// New builds a service that tries each resolver in the order provided.
func New(resolvers ...Resolver) Service {
	return chain(resolvers)
}

type chain []Resolver

// Authenticate resolves credentials by asking each resolver to claim the request.
func (c chain) Authenticate(ctx context.Context, sess *zen.Session) (*Principal, error) {
	for _, resolver := range c {
		principal, err := resolver.Resolve(ctx, sess)
		if err != nil {
			return nil, err
		}
		if principal == nil {
			continue
		}

		sess.WorkspaceID = principal.WorkspaceID
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

// Authorize evaluates the principal's permissions against the query.
func (c chain) Authorize(_ context.Context, principal *Principal, query rbac.PermissionQuery) error {
	if principal == nil {
		return fault.New("missing principal",
			fault.Code(codes.Auth.Authentication.Missing.URN()),
			fault.Internal("principal is nil"),
			fault.Public("You must authenticate before authorizing this request."),
		)
	}

	return rbac.Check(query, principal.Permissions)
}
