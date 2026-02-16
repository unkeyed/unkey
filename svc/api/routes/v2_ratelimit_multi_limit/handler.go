package v2_ratelimit_multi_limit

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitMultiLimitRequestBody
	Response = openapi.V2RatelimitMultiLimitResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit multiLimit endpoint
type Handler struct {
	Keys       keys.KeyService
	ClickHouse clickhouse.Bufferer
	Ratelimit  ratelimit.Service
	Namespaces namespace.Service
	TestMode   bool
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.multiLimit"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Collect unique namespace names
	uniqueNames := make(map[string]bool)
	for _, check := range req {
		uniqueNames[check.Namespace] = true
	}
	names := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		names = append(names, name)
	}

	// Batch load all namespaces
	namespaces, missing, err := h.Namespaces.GetMany(ctx, auth.AuthorizedWorkspaceID, names)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching namespaces."),
		)
	}

	// Auto-create any missing namespaces
	if len(missing) > 0 {
		err = auth.VerifyRootKey(ctx, keys.WithPermissions(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.CreateNamespace,
			}),
		))
		if err != nil {
			return err
		}

		created, createErr := h.Namespaces.CreateMany(ctx, auth.AuthorizedWorkspaceID, missing, &namespace.AuditContext{
			ActorID:   auth.Key.ID,
			ActorName: auth.Key.Name.String,
			ActorType: auditlog.RootKeyActor,
			RemoteIP:  s.Location(),
			UserAgent: s.UserAgent(),
		})
		if createErr != nil {
			return createErr
		}

		for name, ns := range created {
			namespaces[name] = ns
		}
	}

	// Verify permissions for rate limiting
	requiredPerms := make([]rbac.PermissionQuery, 0, len(req))
	for _, check := range req {
		ns := namespaces[check.Namespace]
		requiredPerms = append(requiredPerms, rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   ns.ID,
			Action:       rbac.Limit,
		}))
	}

	wildcardPermission := rbac.T(rbac.Tuple{
		ResourceType: rbac.Ratelimit,
		ResourceID:   "*",
		Action:       rbac.Limit,
	})

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(wildcardPermission, rbac.And(requiredPerms...))))
	if err != nil {
		return err
	}

	// Build ratelimit requests for all checks
	ratelimitReqs := make([]ratelimit.RatelimitRequest, len(req))
	checkMetadata := make([]checkMeta, len(req))

	reqTime := time.Now()
	if h.TestMode {
		header := s.Request().Header.Get("X-Test-Time")
		if header != "" {
			if ts, parseErr := strconv.ParseInt(header, 10, 64); parseErr != nil {
				logger.Warn("invalid test time", "header", header)
			} else {
				reqTime = time.UnixMilli(ts)
			}
		}
	}

	for i, check := range req {
		ns := namespaces[check.Namespace]

		if ns.DeletedAtM.Valid {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.Gone.URN()),
				fault.Public("This namespace has been deleted. Contact support to restore."),
			)
		}

		// Apply override if found, otherwise use request values
		limit, duration, overrideID, matchErr := getLimitAndDuration(check, ns)
		if matchErr != nil {
			return fault.Wrap(matchErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("error matching overrides"),
				fault.Public("Error matching ratelimit override"),
			)
		}

		ratelimitReqs[i] = ratelimit.RatelimitRequest{
			Name:       ns.ID,
			Identifier: check.Identifier,
			Duration:   time.Duration(duration) * time.Millisecond,
			Limit:      limit,
			Cost:       ptr.SafeDeref(check.Cost, 1),
			Time:       reqTime,
		}

		checkMetadata[i] = checkMeta{
			namespaceName: check.Namespace,
			namespaceID:   ns.ID,
			identifier:    check.Identifier,
			overrideID:    overrideID,
			limit:         limit,
		}
	}

	// Batch rate limit all requests using RatelimitMany
	start := time.Now()
	results, err := h.Ratelimit.RatelimitMany(ctx, ratelimitReqs)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("rate limit failed"),
			fault.Public("We're unable to process the rate limit requests."),
		)
	}
	latency := time.Since(start).Milliseconds()

	// Log to ClickHouse if enabled
	if s.ShouldLogRequestToClickHouse() {
		for i, result := range results {
			meta := checkMetadata[i]
			h.ClickHouse.BufferRatelimit(schema.Ratelimit{
				RequestID:   s.RequestID(),
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Time:        start.UnixMilli(),
				NamespaceID: meta.namespaceID,
				Identifier:  meta.identifier,
				Passed:      result.Success,
				Latency:     float64(latency) / float64(len(results)),
				OverrideID:  meta.overrideID,
				Limit:       uint64(result.Limit),
				Remaining:   uint64(result.Remaining),
				ResetAt:     result.Reset.UnixMilli(),
			})
		}
	}

	// Build response and calculate overall success
	limits := make([]openapi.V2RatelimitMultiLimitCheck, len(results))
	allPassed := true
	for i, result := range results {
		meta := checkMetadata[i]
		limits[i] = openapi.V2RatelimitMultiLimitCheck{
			Namespace:  meta.namespaceName,
			Identifier: meta.identifier,
			Passed:     result.Success,
			Limit:      meta.limit,
			Remaining:  result.Remaining,
			Reset:      result.Reset.UnixMilli(),
			OverrideId: meta.overrideID,
		}

		if !result.Success {
			allPassed = false
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitMultiLimitResponseData{
			Passed: allPassed,
			Limits: limits,
		},
	})
}

type checkMeta struct {
	namespaceName string
	namespaceID   string
	identifier    string
	overrideID    string
	limit         int64
}

func getLimitAndDuration(check openapi.V2RatelimitLimitRequestBody, namespace db.FindRatelimitNamespace) (int64, int64, string, error) {
	override, found, err := matchOverride(check.Identifier, namespace)
	if err != nil {
		return 0, 0, "", err
	}

	if found {
		return override.Limit, override.Duration, override.ID, nil
	}

	return check.Limit, check.Duration, "", nil
}

func matchOverride(identifier string, namespace db.FindRatelimitNamespace) (db.FindRatelimitNamespaceLimitOverride, bool, error) {
	if override, ok := namespace.DirectOverrides[identifier]; ok {
		return override, true, nil
	}

	for _, override := range namespace.WildcardOverrides {
		ok, err := match.Wildcard(identifier, override.Identifier)
		if err != nil {
			return db.FindRatelimitNamespaceLimitOverride{}, false, err
		}

		if !ok {
			continue
		}

		return override, true, nil
	}

	return db.FindRatelimitNamespaceLimitOverride{
		Limit:      0,
		ID:         "",
		Identifier: "",
		Duration:   0,
	}, false, nil
}
