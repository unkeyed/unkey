package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	chquery "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
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
	DB                         db.Database
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

// Handle processes the HTTP request.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	wildcard := rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "*", Action: rbac.ReadAnalytics})
	hasWildcard := slices.Contains(principal.Permissions, "api.*.read_analytics")
	allowedAPIIDs := extractAllowedAPIIDs(principal.Permissions)
	if !hasWildcard && len(allowedAPIIDs) == 0 {
		return principal.Authorize(wildcard)
	}

	securityFilters := make([]chquery.SecurityFilter, 0, 1)
	if !hasWildcard {
		keySpaces, fetchErr := h.fetchKeyAuthsByAPIIDs(ctx, principal.WorkspaceID, allowedAPIIDs)
		if fetchErr != nil {
			return fetchErr
		}
		allowedKeySpaceIDs := make([]string, 0, len(keySpaces))
		for _, keySpace := range keySpaces {
			allowedKeySpaceIDs = append(allowedKeySpaceIDs, keySpace.KeyAuthID)
		}
		securityFilters = append(securityFilters, chquery.SecurityFilter{Column: "key_space_id", AllowedValues: allowedKeySpaceIDs})
	}

	verifications, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:           req.Query,
		WorkspaceID:     principal.WorkspaceID,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
		SecurityFilters: securityFilters,
	})
	if err != nil {
		return err
	}
	responseBytes, err := json.Marshal(Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: verifications,
	})
	if err != nil {
		return fault.Wrap(err, fault.Public("Failed to encode query results"))
	}
	if len(responseBytes) > clickhouse.AnalyticsResultBytesMax {
		return fault.New("analytics response byte limit exceeded",
			fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
			fault.Public("Query result exceeds the maximum response size."),
		)
	}
	s.AddHeader("Content-Type", "application/json")
	return s.Send(http.StatusOK, responseBytes)
}

// fetchKeyAuthsByAPIIDs fetches key auth rows for the given API IDs using the cache.
func (h *Handler) fetchKeyAuthsByAPIIDs(ctx context.Context, workspaceID string, apiIDs []string) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
	result := make(map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, len(apiIDs))
	uncached := make([]string, 0, len(apiIDs))
	seen := make(map[string]struct{}, len(apiIDs))
	for _, apiID := range apiIDs {
		if _, ok := seen[apiID]; ok {
			continue
		}
		seen[apiID] = struct{}{}

		key := cache.ScopedKey{WorkspaceID: workspaceID, Key: apiID}
		api, hit := h.Caches.ApiToKeyAuthRow.Get(ctx, key)
		if hit == cache.Hit {
			result[key] = api
		} else if hit == cache.Miss {
			uncached = append(uncached, apiID)
		}
	}

	if len(uncached) == 0 {
		return result, nil
	}

	apis, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), db.FindKeyAuthsByIdsParams{
		WorkspaceID: workspaceID,
		ApiIds:      uncached,
	})
	if err != nil {
		return nil, err
	}

	for _, api := range apis {
		key := cache.ScopedKey{WorkspaceID: workspaceID, Key: api.ApiID}
		result[key] = api
		h.Caches.ApiToKeyAuthRow.Set(ctx, key, api)
	}

	for _, apiID := range uncached {
		key := cache.ScopedKey{WorkspaceID: workspaceID, Key: apiID}
		if _, ok := result[key]; !ok {
			h.Caches.ApiToKeyAuthRow.SetNull(ctx, key)
		}
	}

	return result, nil
}

// extractAllowedAPIIDs extracts API IDs from analytics permissions.
func extractAllowedAPIIDs(permissions []string) []string {
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
