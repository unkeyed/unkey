package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionForbiddenDisabledPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	// Insert a disabled portal config.
	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceID,
		Slug:        "disabled-portal",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     false,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Slug:        "disabled-portal",
		ExternalId:  "user_123",
		Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, 403, res.Status)
	require.NotNil(t, res.Body)
}

// TestCreateSessionForbiddenForNonRootPrincipal guarantees that non-root
// callers cannot probe portal configurations or mint end-user sessions.
func TestCreateSessionForbiddenForNonRootPrincipal(t *testing.T) {
	tests := []struct {
		name          string
		principalType authprincipal.Type
		source        authprincipal.Source
	}{
		{name: "JWT", principalType: authprincipal.TypeJWT, source: authprincipal.JWTSource{}},
		{name: "non-root API key", principalType: authprincipal.TypeAPIKey, source: authprincipal.KeySource{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &handler.Handler{
				DB:            h.DB,
				Auditlogs:     h.Auditlogs,
				PortalBaseURL: "https://portal.unkey.com",
			}

			middlewares := append([]zen.Middleware{}, h.PublicMiddleware()...)
			middlewares = append(middlewares, func(next zen.HandleFunc) zen.HandleFunc {
				return func(ctx context.Context, session *zen.Session) error {
					session.SetPrincipal(&authprincipal.Principal{
						Version: authprincipal.Version,
						Subject: authprincipal.Subject{
							ID:   "user_123",
							Name: "Portal Admin",
							Type: authprincipal.SubjectTypeUser,
						},
						Type:        test.principalType,
						Source:      test.source,
						WorkspaceID: h.Resources().UserWorkspace.ID,
					})
					return next(ctx, session)
				}
			})
			h.Register(route, middlewares...)

			req := handler.Request{
				Slug:        "hidden-portal",
				ExternalId:  "customer_123",
				Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
			}
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token"},
			}

			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusForbidden, res.Status)
			require.Equal(t, "This operation requires a root key.", res.Body.Error.Detail)
			require.NotContains(t, res.RawBody, req.Slug)
			require.NotContains(t, res.RawBody, req.ExternalId)
		})
	}
}
