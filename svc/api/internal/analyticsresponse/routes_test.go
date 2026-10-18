package analyticsresponse_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/openapi"
	gateway "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_gateway_requests"
	ratelimits "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_ratelimits"
	runtimeLogs "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_runtime_logs"
	verifications "github.com/unkeyed/unkey/svc/api/routes/v2_analytics_get_verifications"
)

func TestAnalyticsRoutesCompressResultsButNotSizeErrors(t *testing.T) {
	manager := &resultManager{}
	workspaceID := uid.New(uid.WorkspacePrefix)
	for _, tt := range []struct {
		route zen.Route
		table string
	}{
		{route: &verifications.Handler{AnalyticsConnectionManager: manager}, table: "key_verifications_v1"},
		{route: &ratelimits.Handler{AnalyticsConnectionManager: manager}, table: "ratelimits_v1"},
		{route: &gateway.Handler{AnalyticsConnectionManager: manager}, table: "gateway_requests_v1"},
		{route: &runtimeLogs.Handler{AnalyticsConnectionManager: manager}, table: "runtime_logs_v1"},
	} {
		t.Run(tt.route.Path(), func(t *testing.T) {
			server, err := zen.New(zen.Config{})
			require.NoError(t, err)
			server.RegisterRoute([]zen.Middleware{
				middleware.WithErrorHandling(),
				func(next zen.HandleFunc) zen.HandleFunc {
					return func(ctx context.Context, s *zen.Session) error {
						s.SetPrincipal(&principal.Principal{
							Type:                  principal.TypeAPIKey,
							AuthorizedWorkspaceID: workspaceID,
							Permissions:           []string{"unkey:v1:" + workspaceID + ":**#*"},
						})
						return next(ctx, s)
					}
				},
			}, tt.route)
			for _, encoding := range []string{"", "gzip", "zstd"} {
				t.Run(encoding, func(t *testing.T) {
					manager.rows = []map[string]any{{"region": strings.Repeat("region-", 512)}}
					request, err := json.Marshal(openapi.V2AnalyticsGetVerificationsRequestBody{
						Query: "SELECT region FROM " + tt.table,
					})
					require.NoError(t, err)
					req := httptest.NewRequest(http.MethodPost, tt.route.Path(), bytes.NewReader(request))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Accept-Encoding", encoding)
					rec := httptest.NewRecorder()
					server.Mux().ServeHTTP(rec, req)
					require.Equal(t, http.StatusOK, rec.Code)
					require.Equal(t, encoding, rec.Header().Get("Content-Encoding"))
					var response openapi.V2AnalyticsGetVerificationsResponseBody
					require.NoError(t, json.Unmarshal(decode(t, rec), &response))
					require.Equal(t, manager.rows, response.Data)
					require.NotEmpty(t, response.Meta.RequestId)
				})
			}
			manager.rows = []map[string]any{{"region": strings.Repeat("x", 4*1024*1024+1)}}
			request, err := json.Marshal(openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT region FROM " + tt.table,
			})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, tt.route.Path(), bytes.NewReader(request))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Encoding", "zstd, gzip")
			rec := httptest.NewRecorder()
			server.Mux().ServeHTTP(rec, req)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			require.Empty(t, rec.Header().Get("Content-Encoding"))
		})
	}
}

type resultManager struct {
	rows []map[string]any
}

func (m *resultManager) GetConnection(context.Context, string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	return resultConnection{rows: m.rows}, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseMaxQueryResultRows: 100,
		QuotaLogsRetentionDays:       30,
	}, nil
}

type resultConnection struct {
	clickhouse.ClickHouse
	rows []map[string]any
}

func (c resultConnection) QueryToMaps(context.Context, string, ...any) ([]map[string]any, error) {
	return c.rows, nil
}
