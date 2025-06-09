package permissions

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

func (s *service) Check(ctx context.Context, keyID string, query rbac.PermissionQuery) error {

	ctx, span := tracing.Start(ctx, "permissions.Check")
	defer span.End()

	permissions, err := s.cache.SWR(ctx, keyID, func(ctx context.Context) ([]string, error) {
		return db.Query.FindPermissionsForKey(ctx, s.db.RO(), db.FindPermissionsForKeyParams{
			KeyID: keyID,
		})

	}, func(err error) cache.Op {
		if err == nil {
			return cache.WriteValue
		}
		return cache.Noop

	})

	if err != nil {
		return fault.Wrap(err, fault.Internal("unable to load permissions from db"))
	}

	res, err := s.rbac.EvaluatePermissions(query, permissions)
	if err != nil {
		return fault.New("unable to evaluate permissions",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal(err.Error()),
			fault.Public("Unhandled exception during permission evaluation."),
		)
	}
	if !res.Valid {
		return fault.New("insufficient permissions",
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
			fault.Internal(res.Message),
			fault.Public(res.Message),
		)
	}

	return nil
}
