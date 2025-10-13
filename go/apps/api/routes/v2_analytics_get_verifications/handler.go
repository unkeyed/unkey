package handler

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/analytics"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/array"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2AnalyticsGetVerificationsRequestBody
type Response = openapi.V2AnalyticsGetVerificationsResponseBody
type ResponseData = openapi.V2AnalyticsGetVerificationsResponseData

// Handler implements zen.Route interface for the v2 Analytics get verifications endpoint
type Handler struct {
	Logger                     logging.Logger
	DB                         db.Database
	Keys                       keys.KeyService
	ClickHouse                 clickhouse.ClickHouse
	AnalyticsConnectionManager analytics.ConnectionManager
	Caches                     caches.Caches
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/analytics.getVerifications"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/analytics.getVerifications")

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Get workspace-specific ClickHouse connection and settings first
	conn, settings, err := h.AnalyticsConnectionManager.GetConnection(ctx, auth.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}

	// Build a list of keyAuthIds that the root key has permissions for.
	securityFilters := []chquery.SecurityFilter{}

	allowedAPIIds := extractAllowedAPIIds(auth.Permissions)
	if len(allowedAPIIds) > 0 {
		apis, _, err := h.Caches.ApiToKeyAuthRow.SWRMany(ctx, array.Map(allowedAPIIds, func(apiId string) cache.ScopedKey {
			return cache.ScopedKey{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Key:         apiId,
			}
		}), func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
			apis, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), db.FindKeyAuthsByIdsParams{WorkspaceID: auth.AuthorizedWorkspaceID, ApiIds: allowedAPIIds})
			if err != nil {
				return nil, err
			}

			return array.Reduce(
				apis,
				func(acc map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, api db.FindKeyAuthsByIdsRow) map[cache.ScopedKey]db.FindKeyAuthsByIdsRow {
					acc[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: api.ApiID}] = api
					return acc
				},
				map[cache.ScopedKey]db.FindKeyAuthsByIdsRow{},
			), nil
		}, caches.DefaultFindFirstOp)
		if err != nil {
			return err
		}

		keyAuthIds := make([]string, 0, len(apis))
		for _, api := range apis {
			keyAuthIds = append(keyAuthIds, api.KeyAuthID)
		}

		securityFilters = append(securityFilters, chquery.SecurityFilter{
			Column:        "key_space_id",
			AllowedValues: keyAuthIds,
		})
	}

	parser := chquery.NewParser(chquery.Config{
		WorkspaceID:     auth.AuthorizedWorkspaceID,
		Limit:           int(settings.MaxQueryResultRows),
		SecurityFilters: securityFilters,
		TableAliases: map[string]string{
			"key_verifications_v1":            "default.key_verifications_raw_v2",
			"key_verifications_per_minute_v1": "default.key_verifications_per_minute_v2",
			"key_verifications_per_hour_v1":   "default.key_verifications_per_hour_v2",
			"key_verifications_per_day_v1":    "default.key_verifications_per_day_v2",
			"key_verifications_per_month_v1":  "default.key_verifications_per_month_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
			"default.key_verifications_per_minute_v2",
			"default.key_verifications_per_hour_v2",
			"default.key_verifications_per_day_v2",
			"default.key_verifications_per_month_v2",
		},
	})

	parsedQuery, err := parser.Parse(ctx, req.Query)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid SQL query"),
		)
	}

	// Now we build permission checks based on the key_space_id(s) one specified in the query itself
	// If none are specified, we will just check if there is wildcard read_analytics permission set
	permissionChecks := []rbac.PermissionQuery{
		// Wildcard API analytics access
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAnalytics,
		}),
	}

	keyAuthIds := parser.ExtractColumn("key_space_id")
	if len(keyAuthIds) > 0 {
		keyAuths, _, err := h.Caches.KeyAuthToApiRow.SWRMany(
			ctx,
			array.Map(keyAuthIds, func(keyAuthId string) cache.ScopedKey {
				return cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: keyAuthId}
			}),
			func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow, error) {
				keyAuths, err := db.Query.FindKeyAuthsByKeyAuthIds(ctx, h.DB.RO(), db.FindKeyAuthsByKeyAuthIdsParams{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					KeyAuthIds: array.Map(keys, func(keyAuth cache.ScopedKey) string {
						return keyAuth.Key
					}),
				})

				if err != nil {
					return nil, err
				}

				return array.Reduce(
					keyAuths,
					func(acc map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow, api db.FindKeyAuthsByKeyAuthIdsRow) map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow {
						acc[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: api.KeyAuthID}] = api
						return acc
					},
					map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow{},
				), nil
			},
			caches.DefaultFindFirstOp,
		)
		if err != nil {
			return err
		}

		apiPermissions := make([]rbac.PermissionQuery, 0)
		for _, keyAuth := range keyAuths {
			apiPermissions = append(apiPermissions, rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   keyAuth.ApiID,
				Action:       rbac.ReadAnalytics,
			}))
		}

		permissionChecks = append(permissionChecks, rbac.And(apiPermissions...))
	}

	// Verify user has at least one of: api.*.read_analytics OR (api.<api_id1>.read_analytics AND api.<api_id2>.read_analytics)
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(permissionChecks...)))
	if err != nil {
		return err
	}

	h.Logger.Info("executing query", "original", req.Query, "parsed", parsedQuery)

	// Execute query using workspace connection
	verifications, err := conn.QueryToMaps(ctx, parsedQuery)
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: ResponseData{
			Verifications: verifications,
		},
	})
}

// extractAllowedAPIIds extracts API IDs from permissions
// Returns empty slice if user has wildcard access (api.*.read_analytics)
// Returns specific API IDs if user has limited access (api.api_123.read_analytics, etc.)
func extractAllowedAPIIds(permissions []string) []string {
	if slices.Contains(permissions, "api.*.read_analytics") {
		return nil
	}

	// Extract specific API IDs from permissions like "api.api_123.read_analytics"
	apiIDs := make([]string, 0)
	for _, perm := range permissions {
		pattern := strings.Split(perm, ".")
		if len(pattern) != 3 {
			continue
		}

		if pattern[0] != "api" || pattern[2] != "read_analytics" {
			continue
		}

		apiIDs = append(apiIDs, pattern[1])
	}

	return apiIDs
}
