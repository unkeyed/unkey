package v2RatelimitLimit

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/match"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2RatelimitLimitRequestBody
	Response = openapi.V2RatelimitLimitResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit limit endpoint
type Handler struct {
	Logger                  logging.Logger
	Keys                    keys.KeyService
	DB                      db.Database
	ClickHouse              clickhouse.Bufferer
	Ratelimit               ratelimit.Service
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
	Auditlogs               auditlogs.AuditLogService
	TestMode                bool
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.limit"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	if s.Request().Header.Get("X-Unkey-Metrics") == "disabled" {
		s.DisableClickHouseLogging()
	}

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	cacheKey := cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: req.Namespace}

	loader := func(ctx context.Context) (db.FindRatelimitNamespace, error) {
		result := db.FindRatelimitNamespace{} // nolint:exhaustruct
		var response db.FindRatelimitNamespaceRow
		response, err = db.WithRetryContext(ctx, func() (db.FindRatelimitNamespaceRow, error) {
			return db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Namespace:   req.Namespace,
			})
		})
		if err != nil {
			return result, err
		}

		result = db.FindRatelimitNamespace{
			ID:                response.ID,
			WorkspaceID:       response.WorkspaceID,
			Name:              response.Name,
			CreatedAtM:        response.CreatedAtM,
			UpdatedAtM:        response.UpdatedAtM,
			DeletedAtM:        response.DeletedAtM,
			DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
			WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
		}

		overrides := make([]db.FindRatelimitNamespaceLimitOverride, 0)
		if overrideBytes, ok := response.Overrides.([]byte); ok && overrideBytes != nil {
			err = json.Unmarshal(overrideBytes, &overrides)
			if err != nil {
				return result, err
			}
		}

		for _, override := range overrides {
			result.DirectOverrides[override.Identifier] = override
			if strings.Contains(override.Identifier, "*") {
				result.WildcardOverrides = append(result.WildcardOverrides, override)
			}
		}

		return result, nil
	}

	namespace, hit, err := h.RatelimitNamespaceCache.SWR(
		ctx,
		cacheKey,
		loader,
		caches.DefaultFindFirstOp,
	)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."),
		)
	}

	if hit == cache.Null || db.IsNotFound(err) {
		err = auth.VerifyRootKey(ctx, keys.WithPermissions(
			rbac.T(
				rbac.Tuple{
					ResourceType: rbac.Ratelimit,
					ResourceID:   "*",
					Action:       rbac.CreateNamespace,
				},
			),
		))
		if err != nil {
			return err
		}

		namespace, err = db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.FindRatelimitNamespace, error) {
			//nolint: exhaustruct
			result := db.FindRatelimitNamespace{}
			now := time.Now().UnixMilli()
			id := uid.New(uid.RatelimitNamespacePrefix)

			err = db.Query.InsertRatelimitNamespace(ctx, tx, db.InsertRatelimitNamespaceParams{
				ID:          id,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        req.Namespace,
				CreatedAt:   now,
			})
			if err != nil && !db.IsDuplicateKeyError(err) {
				return result, fault.Wrap(err,
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Public("An unexpected error occurred while creating the namespace."),
				)
			}

			if db.IsDuplicateKeyError(err) {
				namespace, err = loader(ctx)
				if err != nil {
					return result, fault.Wrap(err,
						fault.Code(codes.App.Internal.UnexpectedError.URN()),
						fault.Public("An unexpected error occurred while fetching the namespace."),
					)
				}

				return namespace, err
			}

			result = db.FindRatelimitNamespace{
				ID:                id,
				WorkspaceID:       auth.AuthorizedWorkspaceID,
				Name:              req.Namespace,
				CreatedAtM:        now,
				UpdatedAtM:        sql.NullInt64{Valid: false, Int64: 0},
				DeletedAtM:        sql.NullInt64{Valid: false, Int64: 0},
				DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
				WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
			}

			// Audit log for namespace creation
			err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
				{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitNamespaceCreateEvent,
					Display:     "Created ratelimit namespace " + req.Namespace,
					ActorID:     auth.Key.ID,
					ActorName:   auth.Key.Name.String,
					ActorMeta:   map[string]any{},
					ActorType:   auditlog.RootKeyActor,
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							ID:          id,
							Type:        auditlog.RatelimitNamespaceResourceType,
							Meta:        nil,
							Name:        req.Namespace,
							DisplayName: req.Namespace,
						},
					},
				},
			})
			if err != nil {
				return result, err
			}

			h.RatelimitNamespaceCache.Set(ctx, cacheKey, result)

			return result, nil
		})
		if err != nil {
			return err
		}
	}

	if namespace.DeletedAtM.Valid {
		return fault.New("namespace was deleted",
			fault.Code(codes.Data.RatelimitNamespace.Gone.URN()),
			fault.Public("This namespace has been deleted. Contact support to restore."),
		)
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
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
	)))
	if err != nil {
		return err
	}

	// Apply override if found, otherwise use request values
	limit, duration, overrideID, err := getLimitAndDuration(req, namespace)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("error matching overrides"),
			fault.Public("Error matching ratelimit override"),
		)
	}

	// Apply rate limit
	limitReq := ratelimit.RatelimitRequest{
		Name:       namespace.ID,
		Identifier: req.Identifier,
		Duration:   time.Duration(duration) * time.Millisecond,
		Limit:      limit,
		Cost:       ptr.SafeDeref(req.Cost, 1),
		Time:       time.Time{},
	}

	if h.TestMode {
		header := s.Request().Header.Get("X-Test-Time")
		if header != "" {
			i, parseErr := strconv.ParseInt(header, 10, 64)
			if parseErr != nil {
				h.Logger.Warn("invalid test time", "header", header)
			} else {
				limitReq.Time = time.UnixMilli(i)
			}
		}
	}

	t0 := time.Now()
	result, err := h.Ratelimit.Ratelimit(ctx, limitReq)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("rate limit failed"),
			fault.Public("We're unable to process the rate limit request."),
		)
	}
	latency := time.Since(t0).Milliseconds()
	if s.ShouldLogRequestToClickHouse() {
		h.ClickHouse.BufferRatelimit(schema.Ratelimit{
			RequestID:   s.RequestID(),
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Time:        time.Now().UnixMilli(),
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Passed:      result.Success,
			Latency:     float64(latency),
			OverrideID:  overrideID,
			Limit:       uint64(result.Limit),
			Remaining:   uint64(result.Remaining),
			ResetAt:     result.Reset.UnixMilli(),
		})
	}

	res := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitLimitResponseData{
			Success:    result.Success,
			Limit:      limit,
			Remaining:  result.Remaining,
			Reset:      result.Reset.UnixMilli(),
			OverrideId: overrideID,
		},
	}

	// Return success response
	return s.JSON(http.StatusOK, res)
}

func getLimitAndDuration(req Request, namespace db.FindRatelimitNamespace) (int64, int64, string, error) {
	override, found, err := matchOverride(req.Identifier, namespace)
	if err != nil {
		return 0, 0, "", err
	}

	if found {
		return override.Limit, override.Duration, override.ID, nil
	}

	return req.Limit, req.Duration, "", nil
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
