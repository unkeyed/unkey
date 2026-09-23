package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	chquery "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
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
	hasLegacyWildcard := slices.Contains(principal.Permissions, "api.*.read_analytics")
	allowedAPIIDs := extractAllowedAPIIDs(principal.Permissions)
	canonicalScope := extractCanonicalAnalyticsScope(
		principal.AuthorizedWorkspaceID,
		principal.Permissions,
	)
	if !hasLegacyWildcard && len(allowedAPIIDs) == 0 && !canonicalScope.hasGrant {
		return principal.Authorize(wildcard)
	}

	securityFilters := make([]chquery.SecurityFilter, 0, 1)
	if !hasLegacyWildcard && !canonicalScope.unrestricted {
		keySpaces, fetchErr := h.fetchKeyAuthsByAPIIDs(ctx, principal.AuthorizedWorkspaceID, allowedAPIIDs)
		if fetchErr != nil {
			return fetchErr
		}
		allowedKeySpaces := make(map[string]struct{}, len(keySpaces)+len(canonicalScope.keySpaceIDs))
		for _, keySpace := range keySpaces {
			allowedKeySpaces[keySpace.KeyAuthID] = struct{}{}
		}

		if canonicalScope.hasGrant {
			ownership, err := db.Query.FindKeySpaceAnalyticsOwnership(ctx, h.DB.RO(), db.FindKeySpaceAnalyticsOwnershipParams{
				WorkspaceID: principal.AuthorizedWorkspaceID,
				KeySpaceIds: canonicalScope.keySpaceIDs,
				ProjectIds:  canonicalScope.projectIDs,
			})
			if err != nil {
				return err
			}
			for _, keySpace := range ownership {
				query := rbac.U(
					urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(keySpace.ProjectID).Keyspace(keySpace.ID).Logs(),
					permissions.Read,
				)
				if rbac.Check(query, principal.Permissions) == nil {
					allowedKeySpaces[keySpace.ID] = struct{}{}
				}
			}
		}
		allowedKeySpaceIDs := make([]string, 0, len(allowedKeySpaces))
		for keySpaceID := range allowedKeySpaces {
			allowedKeySpaceIDs = append(allowedKeySpaceIDs, keySpaceID)
		}
		securityFilters = append(securityFilters, chquery.SecurityFilter{Column: "key_space_id", AllowedValues: allowedKeySpaceIDs})
	}

	verifications, err := analytics.Execute(ctx, h.AnalyticsConnectionManager, analytics.ExecuteRequest{
		Query:           req.Query,
		WorkspaceID:     principal.AuthorizedWorkspaceID,
		TableAliases:    tableAliases,
		AllowedTables:   allowedTables,
		SecurityFilters: securityFilters,
		SecurityScopes:  nil,
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

// canonicalAnalyticsScope identifies the ownership rows needed to evaluate
// canonical log grants without treating an empty result as unrestricted.
type canonicalAnalyticsScope struct {
	hasGrant     bool
	unrestricted bool
	keySpaceIDs  []string
	projectIDs   []string
}

// extractCanonicalAnalyticsScope finds candidate ownership rows for canonical
// log grants. The handler validates every candidate against its actual owner.
func extractCanonicalAnalyticsScope(workspaceID string, grantedPermissions []string) canonicalAnalyticsScope {
	scope := canonicalAnalyticsScope{
		hasGrant:     false,
		unrestricted: false,
		keySpaceIDs:  nil,
		projectIDs:   nil,
	}
	for _, grantedPermission := range grantedPermissions {
		resourceName, action, ok := strings.Cut(grantedPermission, "#")
		if !ok || strings.Contains(action, "#") || (action != permissions.Read.String() && action != permissions.Wildcard) {
			continue
		}
		resource, err := urn.ParseV1(resourceName)
		if err != nil || resource.WorkspaceID != workspaceID {
			continue
		}
		if resource.Resource == "**" {
			return canonicalAnalyticsScope{hasGrant: true, unrestricted: true, keySpaceIDs: nil, projectIDs: nil}
		}
		if action == permissions.Wildcard {
			continue
		}

		segments := strings.Split(resource.Resource, "/")
		if len(segments) == 6 && segments[4] == "logs" && segments[5] == "**" {
			segments = segments[:5]
		}
		if len(segments) < 3 || segments[0] != "projects" {
			continue
		}
		projectID := segments[1]
		switch {
		case len(segments) == 3 && segments[2] == "**":
			scope.hasGrant = true
			if projectID == "*" {
				return canonicalAnalyticsScope{hasGrant: true, unrestricted: true, keySpaceIDs: nil, projectIDs: nil}
			}
			scope.projectIDs = append(scope.projectIDs, projectID)
		case len(segments) == 5 && segments[2] == "keyspaces" && (segments[4] == "logs" || segments[4] == "**"):
			scope.hasGrant = true
			keySpaceID := segments[3]
			if projectID == "*" {
				return canonicalAnalyticsScope{hasGrant: true, unrestricted: true, keySpaceIDs: nil, projectIDs: nil}
			}
			if keySpaceID == "*" {
				scope.projectIDs = append(scope.projectIDs, projectID)
			} else {
				scope.keySpaceIDs = append(scope.keySpaceIDs, keySpaceID)
			}
		}
	}

	return scope
}
