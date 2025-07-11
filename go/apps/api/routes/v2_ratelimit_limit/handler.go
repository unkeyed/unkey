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
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/match"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

// Handler implements zen.Route interface for the v2 ratelimit limit endpoint
type Handler struct {
	// Services as public fields
	Logger                        logging.Logger
	Keys                          keys.KeyService
	DB                            db.Database
	ClickHouse                    clickhouse.Bufferer
	Ratelimit                     ratelimit.Service
	RatelimitNamespaceByNameCache cache.Cache[string, db.FindRatelimitNamespace]
	TestMode                      bool
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
	// Authenticate the request with a root key
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// Parse request body
	req := new(Request)
	if bindErr := s.BindBody(req); bindErr != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"), fault.Public("We're unable to parse the request body as JSON."),
		)
	}

	cost := int64(1)
	if req.Cost != nil {
		cost = *req.Cost
	}

	ctx, span := tracing.Start(ctx, "FindRatelimitNamespace")
	namespace, err := h.RatelimitNamespaceByNameCache.SWR(ctx, req.Namespace, func(ctx context.Context) (db.FindRatelimitNamespace, error) {
		response, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Name:        sql.NullString{String: req.Namespace, Valid: true},
		})
		result := db.FindRatelimitNamespace{}
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
		err = json.Unmarshal(response.Overrides.([]byte), &overrides)
		if err != nil {
			return result, err
		}

		for _, override := range overrides {
			result.DirectOverrides[override.Identifier] = override
			if strings.Contains(override.Identifier, "*") {
				result.WildcardOverrides = append(result.WildcardOverrides, override)
			}
		}

		return result, nil
	}, caches.DefaultFindFirstOp)
	span.End()

	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"), fault.Public("This namespace does not exist."),
			)
		}

		return err
	}

	if namespace.DeletedAtM.Valid {
		return fault.New("namespace was deleted",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace not found"), fault.Public("This namespace does not exist."),
		)
	}

	// Verify permissions for rate limiting
	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
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

	// Determine limit and duration from override or request
	var (
		limit      = req.Limit
		duration   = req.Duration
		overrideId = ""
	)

	found, override := matchOverride(req.Identifier, namespace)
	if found {
		limit = int64(override.Limit)
		duration = int64(override.Duration)
		overrideId = override.ID
	}

	// Apply rate limit
	limitReq := ratelimit.RatelimitRequest{
		Identifier: req.Identifier,
		Duration:   time.Duration(duration) * time.Millisecond,
		Limit:      limit,
		Cost:       cost,
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

	result, err := h.Ratelimit.Ratelimit(ctx, limitReq)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("rate limit failed"), fault.Public("We're unable to process the rate limit request."),
		)
	}

	if s.Request().Header.Get("X-Unkey-Metrics") != "disabled" {
		h.ClickHouse.BufferRatelimit(schema.RatelimitRequestV1{
			RequestID:   s.RequestID(),
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Time:        time.Now().UnixMilli(),
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Passed:      result.Success,
		})
	}
	res := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.RatelimitLimitResponseData{
			Success:    result.Success,
			Limit:      limit,
			Remaining:  result.Remaining,
			Reset:      result.Reset.UnixMilli(),
			OverrideId: nil,
		},
	}

	if overrideId != "" {
		res.Data.OverrideId = &overrideId
	}

	// Return success response
	return s.JSON(http.StatusOK, res)
}

func matchOverride(identifier string, namespace db.FindRatelimitNamespace) (bool, db.FindRatelimitNamespaceLimitOverride) {
	// First check for exact match in direct overrides
	if override, ok := namespace.DirectOverrides[identifier]; ok {
		return true, override
	}

	// Then check wildcard overrides
	for _, override := range namespace.WildcardOverrides {
		if match.Wildcard(identifier, override.Identifier) {
			return true, override
		}
	}

	return false, db.FindRatelimitNamespaceLimitOverride{}
}
