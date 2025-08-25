package v2RatelimitLimit

import (
	"context"
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
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitLimitRequestBody
type Response = openapi.V2RatelimitLimitResponseBody

// Handler implements zen.Route interface for the v2 ratelimit limit endpoint
type Handler struct {
	// Services as public fields
	Logger                  logging.Logger
	Keys                    keys.KeyService
	DB                      db.Database
	ClickHouse              clickhouse.Bufferer
	Ratelimit               ratelimit.Service
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
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

	// Authenticate the request with a root key
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	cost := int64(1)
	if req.Cost != nil {
		cost = *req.Cost
	}

	// Use the namespace field directly - it can be either name or ID
	namespaceKey := req.Namespace

	namespace, hit, err := h.RatelimitNamespaceCache.SWR(ctx,
		cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: namespaceKey},
		func(ctx context.Context) (db.FindRatelimitNamespace, error) {
			response, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Namespace:   namespaceKey,
			})
			result := db.FindRatelimitNamespace{} // nolint:exhaustruct
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
		}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Public("This namespace does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."),
		)
	}

	if hit == cache.Null {
		return fault.New("namespace cache null",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Public("This namespace does not exist."),
		)
	}

	if namespace.DeletedAtM.Valid {
		return fault.New("namespace was deleted",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Public("This namespace does not exist."),
		)
	}

	// Verify permissions for rate limiting
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

	// Determine limit and duration from override or request
	var (
		limit      = req.Limit
		duration   = req.Duration
		overrideId = ""
	)

	override, found, err := matchOverride(req.Identifier, namespace)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("error matching overrides"), fault.Public("Error matching ratelimit override"),
		)
	}

	if found {
		limit = override.Limit
		duration = override.Duration
		overrideId = override.ID
	}

	// Apply rate limit
	limitReq := ratelimit.RatelimitRequest{
		Identifier: namespace.ID + ":" + req.Identifier,
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
			fault.Internal("rate limit failed"),
			fault.Public("We're unable to process the rate limit request."),
		)
	}

	if s.ShouldLogRequestToClickHouse() {
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
		Data: openapi.V2RatelimitLimitResponseData{
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

	return db.FindRatelimitNamespaceLimitOverride{}, false, nil
}
