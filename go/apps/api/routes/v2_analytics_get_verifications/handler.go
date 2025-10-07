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
	resulttransformer "github.com/unkeyed/unkey/go/pkg/clickhouse/result-transformer"
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
	// Services as public fields
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

	// Capture apiIds extracted from the query for permission checking
	var extractedAPIIds []string

	// Build security filters for row-level access control
	securityFilters := []chquery.SecurityFilter{}

	// Add API filter if user doesn't have wildcard access
	if allowedAPIIds := extractAllowedAPIIds(auth.Permissions); len(allowedAPIIds) > 0 {
		securityFilters = append(securityFilters, chquery.SecurityFilter{
			Column:        "api_id",
			AllowedValues: allowedAPIIds,
		})
	}

	parser := chquery.NewParser(chquery.Config{
		WorkspaceID:     auth.AuthorizedWorkspaceID,
		Limit:           int(settings.MaxQueryResultRows),
		SecurityFilters: securityFilters,
		TableAliases: map[string]string{
			"key_verifications":            "default.key_verifications_raw_v2",
			"key_verifications_per_minute": "default.key_verifications_per_minute_v2",
			"key_verifications_per_hour":   "default.key_verifications_per_hour_v2",
			"key_verifications_per_day":    "default.key_verifications_per_day_v2",
			"key_verifications_per_month":  "default.key_verifications_per_month_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
			"default.key_verifications_per_minute_v2",
			"default.key_verifications_per_hour_v2",
			"default.key_verifications_per_day_v2",
			"default.key_verifications_per_month_v2",
		},
		VirtualColumns: map[string]chquery.VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Aliases:      []string{"api_id"},
				Resolver: func(ctx context.Context, apiIDs []string) (map[string]string, error) {
					extractedAPIIds = apiIDs

					rowLookup, _, err := h.Caches.ApiToKeyAuthRow.SWRMany(
						ctx,
						array.Map(apiIDs, func(apiID string) cache.ScopedKey {
							return cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: apiID}
						}),
						func(ctx context.Context, missingAPIIds []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
							results, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), db.FindKeyAuthsByIdsParams{
								WorkspaceID: auth.AuthorizedWorkspaceID,
								ApiIds: array.Map(missingAPIIds, func(key cache.ScopedKey) string {
									return key.Key
								}),
							})
							if err != nil {
								return nil, fault.Wrap(err,
									fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
									fault.Public("Failed to lookup APIs"),
								)
							}

							apiToRow := make(map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, len(results))
							keyAuthToRow := make(map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, len(results))
							for _, result := range results {
								apiToRow[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.ApiID}] = result
								keyAuthToRow[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.KeyAuthID}] = result
							}

							// Double-write: cache the reverse mapping too
							h.Caches.KeyAuthToApiRow.SetMany(ctx, keyAuthToRow)

							return apiToRow, nil
						}, func(err error) cache.Op {
							if db.IsNotFound(err) {
								return cache.WriteNull
							}

							if err != nil {
								return cache.Noop
							}

							return cache.WriteValue
						})

					if err != nil {
						return nil, err
					}

					// Convert to simple string map for the resolver
					lookup := make(map[string]string, len(rowLookup))
					for apiID, row := range rowLookup {
						lookup[apiID.Key] = row.KeyAuthID
					}

					// Verify all requested API IDs were found
					for _, apiID := range apiIDs {
						_, found := lookup[apiID]

						if found {
							continue
						}

						return nil, fault.New("api not found",
							fault.Code(codes.Data.Api.NotFound.URN()),
							fault.Public("One or more API IDs do not exist"),
						)
					}

					return lookup, nil
				},
			},
			"externalId": {
				ActualColumn: "identity_id",
				Aliases:      []string{"external_id"},
				Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
					identityLookup, _, err := h.Caches.ExternalIdToIdentity.SWRMany(
						ctx,
						array.Map(externalIDs, func(externalID string) cache.ScopedKey {
							return cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: externalID}
						}),
						func(ctx context.Context, missingScopedKeys []cache.ScopedKey) (map[cache.ScopedKey]db.Identity, error) {
							// Extract external IDs from scoped keys
							missingExternalIDs := make([]string, len(missingScopedKeys))
							for i, sk := range missingScopedKeys {
								missingExternalIDs[i] = sk.Key
							}

							identities, err := db.Query.FindIdentities(ctx, h.DB.RO(), db.FindIdentitiesParams{
								WorkspaceID: auth.AuthorizedWorkspaceID,
								Deleted:     false,
								Identities:  missingExternalIDs,
							})
							if err != nil {
								return nil, fault.Wrap(err,
									fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
									fault.Public("Failed to lookup identities"),
								)
							}

							externalToIdentity := make(map[cache.ScopedKey]db.Identity, len(identities))
							identityByIDCache := make(map[cache.ScopedKey]db.Identity, len(identities))
							for _, identity := range identities {
								externalToIdentity[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: identity.ExternalID}] = identity
								identityByIDCache[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: identity.ID}] = identity
							}

							// Double-write: cache the identities by ID too
							h.Caches.Identity.SetMany(ctx, identityByIDCache)

							return externalToIdentity, nil
						}, func(err error) cache.Op {
							if db.IsNotFound(err) {
								return cache.WriteNull
							}

							if err != nil {
								return cache.Noop
							}

							return cache.WriteValue
						})

					if err != nil {
						return nil, err
					}

					// Convert scoped lookup back to simple map for the resolver
					simpleLookup := make(map[string]string, len(identityLookup))
					for scopedKey, identity := range identityLookup {
						simpleLookup[scopedKey.Key] = identity.ID
					}

					// Verify all requested external IDs were found
					for _, externalID := range externalIDs {
						if _, found := simpleLookup[externalID]; !found {
							return nil, fault.New("identity not found",
								fault.Code(codes.Data.Identity.NotFound.URN()),
								fault.Public("One or more external IDs do not exist"),
							)
						}
					}

					return simpleLookup, nil
				},
			},
		},
	})

	parsedQuery, err := parser.Parse(ctx, req.Query)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid SQL query"),
		)
	}

	h.Logger.Info("executing query", "original", req.Query, "parsed", parsedQuery.Query, "mappings", parsedQuery.ColumnMappings)

	// api.*.read_analytics OR (any specific API permissions api.api_123.read_analytics)
	permissionChecks := []rbac.PermissionQuery{
		// Wildcard API analytics access
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAnalytics,
		}),
	}

	// If query filters by apiId, add specific API permissions check (requires ALL)
	if len(extractedAPIIds) > 0 {
		apiPermissions := make([]rbac.PermissionQuery, len(extractedAPIIds))
		for i, apiID := range extractedAPIIds {
			apiPermissions[i] = rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   apiID,
				Action:       rbac.ReadAnalytics,
			})
		}

		permissionChecks = append(permissionChecks, rbac.And(apiPermissions...))
	}

	// Verify user has at least one of: api.*.read_analytics, or all specific API permissions
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(permissionChecks...)))
	if err != nil {
		return err
	}

	// Execute query using workspace connection
	verifications, err := conn.QueryToMaps(ctx, parsedQuery.Query)
	if err != nil {
		return err
	}

	// Configure all possible transformations - transformer will only process columns that exist in results
	transformer := resulttransformer.New([]resulttransformer.ColumnConfig{
		{
			ActualColumn:  "key_space_id",
			VirtualColumn: "apiId",
			Resolver: func(ctx context.Context, keySpaceIDs []string) (map[string]string, error) {
				rowMapping, _, err := h.Caches.KeyAuthToApiRow.SWRMany(
					ctx,
					array.Map(keySpaceIDs, func(keySpaceID string) cache.ScopedKey {
						return cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: keySpaceID}
					}),
					func(ctx context.Context, missingKeyAuthIDs []cache.ScopedKey) (map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, error) {
						results, err := db.Query.FindKeyAuthsByKeyAuthIds(ctx, h.DB.RO(), db.FindKeyAuthsByKeyAuthIdsParams{
							WorkspaceID: auth.AuthorizedWorkspaceID,
							KeyAuthIds: array.Map(missingKeyAuthIDs, func(keyAuthID cache.ScopedKey) string {
								return keyAuthID.Key
							}),
						})
						if err != nil {
							return nil, fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Public("Failed to resolve API IDs"),
							)
						}

						keyAuthToRow := make(map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, len(results))
						apiToRow := make(map[cache.ScopedKey]db.FindKeyAuthsByIdsRow, len(results))
						for _, result := range results {
							row := db.FindKeyAuthsByIdsRow{KeyAuthID: result.KeyAuthID, ApiID: result.ApiID}
							keyAuthToRow[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.KeyAuthID}] = row
							apiToRow[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.ApiID}] = row
						}

						h.Caches.ApiToKeyAuthRow.SetMany(ctx, apiToRow)

						return keyAuthToRow, nil
					}, caches.DefaultFindFirstOp)
				if err != nil {
					return nil, err
				}

				mapping := make(map[string]string, len(rowMapping))
				for keyAuthID, row := range rowMapping {
					mapping[keyAuthID.Key] = row.ApiID
				}

				return mapping, nil
			},
		},
		{
			ActualColumn:  "identity_id",
			VirtualColumn: "externalId",
			Resolver: func(ctx context.Context, identityIDs []string) (map[string]string, error) {
				identityMapping, _, err := h.Caches.Identity.SWRMany(ctx,
					array.Map(identityIDs, func(identityID string) cache.ScopedKey {
						return cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: identityID}
					}),
					func(ctx context.Context, missingIdentityIDs []cache.ScopedKey) (map[cache.ScopedKey]db.Identity, error) {
						identities, err := db.Query.FindIdentities(ctx, h.DB.RO(), db.FindIdentitiesParams{
							WorkspaceID: auth.AuthorizedWorkspaceID,
							Deleted:     false,
							Identities: array.Map(missingIdentityIDs, func(missingIdentityID cache.ScopedKey) string {
								return missingIdentityID.Key
							}),
						})
						if err != nil {
							return nil, fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Public("Failed to resolve identity IDs"),
							)
						}

						identityByIDCache := make(map[cache.ScopedKey]db.Identity, len(identities))
						externalToIdentity := make(map[cache.ScopedKey]db.Identity, len(identities))
						for _, identity := range identities {
							identityByIDCache[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: identity.ID}] = identity
							externalToIdentity[cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: identity.ExternalID}] = identity
						}

						// Double-write: cache the externalId -> identity mapping too
						h.Caches.ExternalIdToIdentity.SetMany(ctx, externalToIdentity)

						return identityByIDCache, nil
					}, caches.DefaultFindFirstOp)

				if err != nil {
					return nil, err
				}

				// Convert Identity objects to simple map
				mapping := make(map[string]string, len(identityMapping))
				for identityID, identity := range identityMapping {
					mapping[identityID.Key] = identity.ExternalID
				}

				return mapping, nil
			},
		},
	})

	// Transform results to user-facing format
	transformedVerifications, err := transformer.TransformWithMappings(ctx, verifications, parsedQuery.ColumnMappings)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("Failed to transform query results"),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: ResponseData{
			Verifications: transformedVerifications,
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
