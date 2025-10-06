package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/analytics"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
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
	AnalyticsConnectionManager *analytics.ConnectionManager
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

	// Check if analytics is configured
	if h.AnalyticsConnectionManager == nil {
		return fault.New("workspace analytics connection manager not initialized",
			fault.Code(codes.Data.Analytics.NotConfigured.URN()),
			fault.Public("Analytics are not configured for this instance"),
		)
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

	// Get workspace-specific ClickHouse connection and settings first
	conn, settings, err := h.AnalyticsConnectionManager.GetConnection(ctx, auth.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}

	// Capture apiIds extracted from the query for permission checking
	var extractedAPIIds []string

	parser := chquery.NewParser(chquery.Config{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Limit:       int(settings.MaxQueryResultRows),
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
					start := time.Now()
					defer func() {
						h.Logger.Info("virtual column resolver: apiId",
							"duration_ms", time.Since(start).Milliseconds(),
							"key_count", len(apiIDs),
						)
					}()

					// Capture the apiIds for permission checking later
					extractedAPIIds = apiIDs

					// Use cache with SWR - cache full row objects
					rowLookup, _, err := h.Caches.ApiToKeyAuthRow.SWRMany(ctx, apiIDs, func(ctx context.Context, missingAPIIds []string) (map[string]db.FindKeyAuthsByIdsRow, error) {
						results, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), db.FindKeyAuthsByIdsParams{
							WorkspaceID: auth.AuthorizedWorkspaceID,
							ApiIds:      missingAPIIds,
						})
						if err != nil {
							return nil, fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Public("Failed to lookup APIs"),
							)
						}

						apiToRow := make(map[string]db.FindKeyAuthsByIdsRow, len(results))
						keyAuthToRow := make(map[string]db.FindKeyAuthsByIdsRow, len(results))
						for _, result := range results {
							apiToRow[result.ApiID] = result
							keyAuthToRow[result.KeyAuthID] = result
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
						lookup[apiID] = row.KeyAuthID
					}

					// Verify all requested API IDs were found
					for _, apiID := range apiIDs {
						if _, found := lookup[apiID]; !found {
							return nil, fault.New("api not found",
								fault.Code(codes.Data.Api.NotFound.URN()),
								fault.Public("One or more API IDs do not exist"),
							)
						}
					}

					return lookup, nil
				},
			},
			"externalId": {
				ActualColumn: "identity_id",
				Aliases:      []string{"external_id"},
				Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
					start := time.Now()
					defer func() {
						h.Logger.Info("virtual column resolver: externalId",
							"duration_ms", time.Since(start).Milliseconds(),
							"key_count", len(externalIDs),
						)
					}()

					// Build scoped keys for cache lookups
					scopedKeys := make([]cache.ScopedKey, len(externalIDs))
					for i, externalID := range externalIDs {
						scopedKeys[i] = cache.ScopedKey{
							WorkspaceID: auth.AuthorizedWorkspaceID,
							Key:         externalID,
						}
					}

					// Use cache with SWR - cache full Identity objects
					identityLookup, _, err := h.Caches.ExternalIdToIdentity.SWRMany(ctx, scopedKeys, func(ctx context.Context, missingScopedKeys []cache.ScopedKey) (map[cache.ScopedKey]db.Identity, error) {
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
						identityByIDCache := make(map[string]db.Identity, len(identities))
						for _, identity := range identities {
							scopedKey := cache.ScopedKey{
								WorkspaceID: auth.AuthorizedWorkspaceID,
								Key:         identity.ExternalID,
							}
							externalToIdentity[scopedKey] = identity
							identityByIDCache[identity.ID] = identity
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

	parseResult, err := parser.Parse(ctx, req.Query)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid SQL query"),
		)
	}

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
	verifications, err := conn.QueryToMaps(ctx, parseResult.Query)
	if err != nil {
		return err
	}

	// Configure all possible transformations - transformer will only process columns that exist in results
	transformer := resulttransformer.New([]resulttransformer.ColumnConfig{
		{
			ActualColumn:  "key_space_id",
			VirtualColumn: "apiId",
			Resolver: func(ctx context.Context, keySpaceIDs []string) (map[string]string, error) {
				rowMapping, _, err := h.Caches.KeyAuthToApiRow.SWRMany(ctx, keySpaceIDs, func(ctx context.Context, missingKeyAuthIDs []string) (map[string]db.FindKeyAuthsByIdsRow, error) {
					results, err := db.Query.FindKeyAuthsByKeyAuthIds(ctx, h.DB.RO(), db.FindKeyAuthsByKeyAuthIdsParams{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						KeyAuthIds:  missingKeyAuthIDs,
					})
					if err != nil {
						return nil, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Public("Failed to resolve API IDs"),
						)
					}

					keyAuthToRow := make(map[string]db.FindKeyAuthsByIdsRow, len(results))
					apiToRow := make(map[string]db.FindKeyAuthsByIdsRow, len(results))
					for _, result := range results {
						row := db.FindKeyAuthsByIdsRow{
							KeyAuthID: result.KeyAuthID,
							ApiID:     result.ApiID,
						}
						keyAuthToRow[result.KeyAuthID] = row
						apiToRow[result.ApiID] = row
					}

					h.Caches.ApiToKeyAuthRow.SetMany(ctx, apiToRow)

					return keyAuthToRow, nil
				}, caches.DefaultFindFirstOp)
				if err != nil {
					return nil, err
				}

				mapping := make(map[string]string, len(rowMapping))
				for keyAuthID, row := range rowMapping {
					mapping[keyAuthID] = row.ApiID
				}

				return mapping, nil
			},
		},
		{
			ActualColumn:  "identity_id",
			VirtualColumn: "externalId",
			Resolver: func(ctx context.Context, identityIDs []string) (map[string]string, error) {
				identityMapping, _, err := h.Caches.Identity.SWRMany(ctx, identityIDs, func(ctx context.Context, missingIdentityIDs []string) (map[string]db.Identity, error) {
					identities, err := db.Query.FindIdentities(ctx, h.DB.RO(), db.FindIdentitiesParams{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Deleted:     false,
						Identities:  missingIdentityIDs,
					})
					if err != nil {
						return nil, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Public("Failed to resolve identity IDs"),
						)
					}

					identityByIDCache := make(map[string]db.Identity, len(identities))
					externalToIdentity := make(map[cache.ScopedKey]db.Identity, len(identities))
					for _, identity := range identities {
						identityByIDCache[identity.ID] = identity
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
					mapping[identityID] = identity.ExternalID
				}

				return mapping, nil
			},
		},
	})

	// Transform results to user-facing format
	transformedVerifications, err := transformer.TransformWithMappings(ctx, verifications, parseResult.ColumnMappings)
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
