package v2RatelimitLimit

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	sf "github.com/unkeyed/unkey/pkg/singleflight"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitLimitRequestBody
	Response = openapi.V2RatelimitLimitResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit limit endpoint
type Handler struct {
	DB             db.Database
	Keys           keys.KeyService
	ClickHouse     clickhouse.Bufferer
	Ratelimit      ratelimit.Service
	NamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
	Auditlogs      auditlogs.AuditLogService
	TestMode       bool
	createFlight   sf.Group[db.FindRatelimitNamespace]
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

	ns, found, err := h.getNamespace(ctx, auth.AuthorizedWorkspaceID, req.Namespace)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."),
		)
	}

	if !found {
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

		ns, err = h.createNamespace(ctx, s, auth, req.Namespace)
		if err != nil {
			return err
		}
	}

	if ns.DeletedAtM.Valid {
		return fault.New("namespace was deleted",
			fault.Code(codes.Data.RatelimitNamespace.Gone.URN()),
			fault.Public("This namespace has been deleted. Contact support to restore."),
		)
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   ns.ID,
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
	limit, duration, overrideID, err := getLimitAndDuration(req, ns)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("error matching overrides"),
			fault.Public("Error matching ratelimit override"),
		)
	}

	// Apply rate limit
	limitReq := ratelimit.RatelimitRequest{
		Name:       ns.ID,
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
				logger.Warn("invalid test time", "header", header)
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
			NamespaceID: ns.ID,
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

func (h *Handler) getNamespace(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error) {
	cacheKey := cache.ScopedKey{WorkspaceID: workspaceID, Key: nameOrID}

	ns, hit, err := h.NamespaceCache.SWR(ctx, cacheKey, func(ctx context.Context) (db.FindRatelimitNamespace, error) {
		row, dbErr := db.WithRetryContext(ctx, func() (db.FindRatelimitNamespaceRow, error) {
			return db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
				WorkspaceID: workspaceID,
				Namespace:   nameOrID,
			})
		})
		if dbErr != nil {
			return db.FindRatelimitNamespace{}, dbErr //nolint:exhaustruct
		}
		return namespace.ParseNamespaceRow(row), nil
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
		}
		return db.FindRatelimitNamespace{}, false, err //nolint:exhaustruct
	}

	if hit == cache.Null {
		return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
	}

	return ns, true, nil
}

func (h *Handler) createNamespace(ctx context.Context, s *zen.Session, auth *keys.KeyVerifier, name string) (db.FindRatelimitNamespace, error) {
	key := auth.AuthorizedWorkspaceID + ":" + name
	return h.createFlight.Do(key, func() (db.FindRatelimitNamespace, error) {
		ns, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.FindRatelimitNamespace, error) {
			now := time.Now().UnixMilli()
			id := uid.New(uid.RatelimitNamespacePrefix)

			insertErr := db.Query.InsertRatelimitNamespace(ctx, tx, db.InsertRatelimitNamespaceParams{
				ID:          id,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        name,
				CreatedAt:   now,
			})
			if insertErr != nil && !db.IsDuplicateKeyError(insertErr) {
				return db.FindRatelimitNamespace{}, fault.Wrap(insertErr, //nolint:exhaustruct
					fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Public("An unexpected error occurred while creating the namespace."),
				)
			}

			if db.IsDuplicateKeyError(insertErr) {
				// Another request created it first — re-fetch using the write connection
				row, fetchErr := db.Query.FindRatelimitNamespace(ctx, tx, db.FindRatelimitNamespaceParams{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Namespace:   name,
				})
				if fetchErr != nil {
					return db.FindRatelimitNamespace{}, fetchErr //nolint:exhaustruct
				}
				return namespace.ParseNamespaceRow(row), nil
			}

			result := db.FindRatelimitNamespace{
				ID:                id,
				WorkspaceID:       auth.AuthorizedWorkspaceID,
				Name:              name,
				CreatedAtM:        now,
				UpdatedAtM:        sql.NullInt64{Valid: false, Int64: 0},
				DeletedAtM:        sql.NullInt64{Valid: false, Int64: 0},
				DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
				WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
			}

			auditErr := h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
				{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.RatelimitNamespaceCreateEvent,
					Display:     "Created ratelimit namespace " + name,
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
							Name:        name,
							DisplayName: name,
						},
					},
				},
			})
			if auditErr != nil {
				return result, auditErr
			}

			return result, nil
		})
		if err != nil {
			return ns, err
		}

		// Warm cache by both name and ID after the transaction has committed
		h.NamespaceCache.Set(ctx, cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: ns.Name}, ns)
		h.NamespaceCache.Set(ctx, cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: ns.ID}, ns)

		return ns, nil
	})
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
