package v2RatelimitLimit

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

type Services struct {
	Logger      logging.Logger
	Keys        keys.KeyService
	DB          db.Database
	Permissions permissions.PermissionService
	Ratelimit   ratelimit.Service
}

// New creates a new route handler for ratelimits.limit
func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/ratelimit.limit", func(ctx context.Context, s *zen.Session) error {
		// Authenticate the request with a root key
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// Parse request body
		req := new(Request)
		if bindErr := s.BindBody(req); bindErr != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("invalid request body", "We're unable to parse the request body as JSON."),
			)
		}

		cost := int64(1)
		if req.Cost != nil {
			cost = *req.Cost
		}

		namespace, err := db.Query.FindRatelimitNamespaceByName(ctx, svc.DB.RO(), db.FindRatelimitNamespaceByNameParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Name:        req.Namespace,
		})
		if err != nil {
			return db.HandleErr(err, "namespace")
		}
		if namespace.DeletedAtM.Valid {
			return fault.New("namespace was deleted",
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("namespace not found", "This namespace does not exist."),
			)
		}

		// Verify permissions for rate limiting
		permission, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   namespace.ID,
					Action:       rbac.Limit,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.Limit,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permission.Valid {
			return fault.New("insufficient permissions",
				fault.WithTag(fault.INSUFFICIENT_PERMISSIONS),
				fault.WithDesc(permission.Message, permission.Message),
			)
		}

		overrides, err := db.Query.FindRatelimitOverrideMatches(ctx, svc.DB.RO(), db.FindRatelimitOverrideMatchesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})
		if err != nil {
			return db.HandleErr(err, "override")
		}

		// Determine limit and duration from override or request
		var (
			limit      = req.Limit
			duration   = req.Duration
			overrideId = ""
		)
		for _, override := range overrides {
			if override.DeletedAtM.Valid {
				continue
			}
			limit = int64(override.Limit)
			duration = int64(override.Duration)
			overrideId = override.ID

			if override.Identifier == req.Identifier {
				// Exact match takes precedence
				break
			}
		}

		// Apply rate limit
		limitReq := ratelimit.RatelimitRequest{
			Identifier: req.Identifier,
			Duration:   time.Duration(duration) * time.Millisecond,
			Limit:      limit,
			Cost:       cost,
		}

		result, err := svc.Ratelimit.Ratelimit(ctx, limitReq)
		if err != nil {
			return fault.Wrap(err,
				fault.WithTag(fault.INTERNAL_SERVER_ERROR),
				fault.WithDesc("rate limit failed", "We're unable to process the rate limit request."),
			)
		}

		res := Response{
			Success:    result.Success,
			Limit:      limit,
			Remaining:  result.Remaining,
			Reset:      result.Reset,
			OverrideId: nil,
		}
		if overrideId != "" {
			res.OverrideId = &overrideId
		}
		// Return success response
		return s.JSON(http.StatusOK, res)
	})
}
