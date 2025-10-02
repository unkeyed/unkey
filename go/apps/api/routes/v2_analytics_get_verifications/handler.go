package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query"
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
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	ClickHouse clickhouse.ClickHouse
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

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAPI,
		}),
	)))
	if err != nil {
		return err
	}

	rewriter := chquery.New(chquery.Config{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Limit:       10_000,
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
				Resolver: func(ctx context.Context, apiIDs []string) (map[string]string, error) {
					// Batch lookup APIs -> key_auth_ids
					results, err := db.Query.FindKeyAuthsByIds(ctx, h.DB.RO(), apiIDs)
					if err != nil {
						return nil, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Public("Failed to lookup APIs"),
						)
					}

					// Build lookup map and verify we found all APIs
					lookup := make(map[string]string)
					for _, result := range results {
						lookup[result.ApiID] = result.KeyAuthID
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

					// TODO: Add workspace_id filter to FindKeyAuthsByIds query to verify workspace ownership in single query
					// For now, we rely on the fact that the query only returns non-deleted APIs

					return lookup, nil
				},
			},
			"externalId": {
				ActualColumn: "identity_id",
				Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
					identities, err := db.Query.FindIdentities(ctx, h.DB.RO(), db.FindIdentitiesParams{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Deleted:     false,
						Identities:  externalIDs,
					})
					if err != nil {
						return nil, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Public("Failed to lookup identities"),
						)
					}

					// Build lookup map
					lookup := make(map[string]string)
					for _, identity := range identities {
						lookup[identity.ExternalID] = identity.ID
					}

					// Verify all requested external IDs were found
					for _, externalID := range externalIDs {
						if _, found := lookup[externalID]; !found {
							return nil, fault.New("identity not found",
								fault.Code(codes.Data.Identity.NotFound.URN()),
								fault.Public("One or more external IDs do not exist"),
							)
						}
					}

					return lookup, nil
				},
			},
		},
	})

	// Rewrite the query (extracts/resolves virtual columns, validates, injects workspace filter)
	safeQuery, err := rewriter.Rewrite(ctx, req.Query)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid SQL query"),
		)
	}

	h.Logger.Info("executing analytics query",
		"requestId", s.RequestID(),
		"workspaceId", auth.AuthorizedWorkspaceID,
		"originalQuery", req.Query,
		"rewrittenQuery", safeQuery,
	)

	// Execute query and scan results into dynamic maps
	verifications, err := h.ClickHouse.QueryToMaps(ctx, safeQuery)
	if err != nil {
		h.Logger.Error("clickhouse query failed",
			"error", err,
			"query", safeQuery,
		)
		return err // Already fault-wrapped with appropriate code by QueryToMaps
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
