package handler

import (
	"context"
	"fmt"
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

var (
	tableAliases = map[string]string{
		"key_verifications_v1":            "default.key_verifications_raw_v2",
		"key_verifications_per_minute_v1": "default.key_verifications_per_minute_v3",
		"key_verifications_per_hour_v1":   "default.key_verifications_per_hour_v3",
		"key_verifications_per_day_v1":    "default.key_verifications_per_day_v3",
		"key_verifications_per_month_v1":  "default.key_verifications_per_month_v3",
	}

	allowedTables = []string{
		"default.key_verifications_raw_v2",
		"default.key_verifications_per_minute_v3",
		"default.key_verifications_per_hour_v3",
		"default.key_verifications_per_day_v3",
		"default.key_verifications_per_month_v3",
	}
)

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

	// Build a list of keySpaceIds that the root key has permissions for.
	securityFilters, err := h.buildSecurityFilters(ctx, auth)
	if err != nil {
		return err
	}

	parser := chquery.NewParser(chquery.Config{
		WorkspaceID:     auth.AuthorizedWorkspaceID,
		Limit:           int(settings.MaxQueryResultRows),
		SecurityFilters: securityFilters,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
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

	keySpaceIds := parser.ExtractColumn("key_space_id")
	if len(keySpaceIds) > 0 {
		apiPermissions, err := h.buildAPIPermissionsFromKeySpaces(ctx, auth, keySpaceIds)
		if err != nil {
			return err
		}
		permissionChecks = append(permissionChecks, rbac.And(apiPermissions...))
	}

	// Verify user has at least one of: api.*.read_analytics OR (api.<api_id1>.read_analytics AND api.<api_id2>.read_analytics)
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(permissionChecks...)))
	if err != nil {
		return err
	}

	h.Logger.Debug("executing query", "original", req.Query, "parsed", parsedQuery)

	// Execute query using workspace connection
	verifications, err := conn.QueryToMaps(ctx, parsedQuery)
	if err != nil {
		return clickhouse.WrapClickHouseError(err)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: verifications,
	})
}

// buildSecurityFilters creates ClickHouse security filters based on user permissions.
// Returns filters that restrict queries to only the key_space_ids the user has access to.
func (h *Handler) buildSecurityFilters(ctx context.Context, auth *keys.KeyVerifier) ([]chquery.SecurityFilter, error) {
	allowedAPIIds := extractAllowedAPIIds(auth.Permissions)
	if len(allowedAPIIds) == 0 {
		return []chquery.SecurityFilter{}, nil
	}

	// Fetch key auths for the allowed API IDs
	apis, err := h.fetchKeyAuthsByAPIIds(ctx, auth.AuthorizedWorkspaceID, allowedAPIIds)
	if err != nil {
		return nil, err
	}

	// Extract key space IDs from the fetched APIs
	keySpaceIds := make([]string, 0, len(apis))
	for _, api := range apis {
		keySpaceIds = append(keySpaceIds, api.KeyAuthID)
	}

	return []chquery.SecurityFilter{
		{
			Column:        "key_space_id",
			AllowedValues: keySpaceIds,
		},
	}, nil
}

// fetchKeyAuthsByAPIIds fetches key auth rows for the given API IDs using the cache.
func (h *Handler) fetchKeyAuthsByAPIIds(ctx context.Context, workspaceID string, apiIDs []string) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
	cacheKeys := array.Map(apiIDs, func(apiID string) cache.ScopedKey {
		return cache.ScopedKey{
			WorkspaceID: workspaceID,
			Key:         apiID,
		}
	})

	apis, _, err := h.Caches.ApiToKeyAuthRow.SWRMany(
		ctx,
		cacheKeys,
		func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
			apis, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), db.FindKeyAuthsByIdsParams{
				WorkspaceID: workspaceID,
				ApiIds:      apiIDs,
			})
			if err != nil {
				return nil, err
			}

			return array.Reduce(
				apis,
				func(acc map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, api db.FindKeyAuthsByIdsRow) map[cache.ScopedKey]db.FindKeyAuthsByIdsRow {
					acc[cache.ScopedKey{WorkspaceID: workspaceID, Key: api.ApiID}] = api
					return acc
				},
				map[cache.ScopedKey]db.FindKeyAuthsByIdsRow{},
			), nil
		},
		caches.DefaultFindFirstOp,
	)

	return apis, err
}

// buildAPIPermissionsFromKeySpaces fetches key spaces and builds RBAC permissions for them.
// Returns an error if any key space is not found.
func (h *Handler) buildAPIPermissionsFromKeySpaces(ctx context.Context, auth *keys.KeyVerifier, keySpaceIds []string) ([]rbac.PermissionQuery, error) {
	keySpaces, keySpaceHits, err := h.fetchKeyAuthsByKeyAuthIds(ctx, auth.AuthorizedWorkspaceID, keySpaceIds)
	if err != nil {
		return nil, err
	}

	// Check for missing key_space_ids and build permissions
	apiPermissions := make([]rbac.PermissionQuery, 0, len(keySpaceHits))
	for key, hit := range keySpaceHits {
		if hit == cache.Null {
			return nil, fault.New("key_space_id not found",
				fault.Code(codes.Data.KeySpace.NotFound.URN()),
				fault.Public(fmt.Sprintf("KeySpace '%s' was not found.", key.Key)),
			)
		}

		apiPermissions = append(apiPermissions, rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   keySpaces[key].ApiID,
			Action:       rbac.ReadAnalytics,
		}))
	}

	return apiPermissions, nil
}

// fetchKeyAuthsByKeyAuthIds fetches key auth rows for the given key auth IDs using the cache.
func (h *Handler) fetchKeyAuthsByKeyAuthIds(ctx context.Context, workspaceID string, keyAuthIDs []string) (map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow, map[cache.ScopedKey]cache.CacheHit, error) {
	cacheKeys := array.Map(keyAuthIDs, func(keyAuthID string) cache.ScopedKey {
		return cache.ScopedKey{
			WorkspaceID: workspaceID,
			Key:         keyAuthID,
		}
	})

	return h.Caches.KeyAuthToApiRow.SWRMany(
		ctx,
		cacheKeys,
		func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow, error) {
			keySpaces, err := db.Query.FindKeyAuthsByKeyAuthIds(ctx, h.DB.RO(), db.FindKeyAuthsByKeyAuthIdsParams{
				WorkspaceID: workspaceID,
				KeyAuthIds: array.Map(keys, func(keySpace cache.ScopedKey) string {
					return keySpace.Key
				}),
			})
			if err != nil {
				return nil, err
			}

			return array.Reduce(
				keySpaces,
				func(acc map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow, api db.FindKeyAuthsByKeyAuthIdsRow) map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow {
					acc[cache.ScopedKey{WorkspaceID: workspaceID, Key: api.KeyAuthID}] = api
					return acc
				},
				map[cache.ScopedKey]db.FindKeyAuthsByKeyAuthIdsRow{},
			), nil
		},
		caches.DefaultFindFirstOp,
	)
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
