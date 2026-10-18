package routes_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes"
)

func TestAnalyticsRoutesCompressResultsButNotSizeErrors(t *testing.T) {
	manager := &resultManager{}
	workspaceID := uid.New(uid.WorkspacePrefix)
	validator, err := validation.New()
	require.NoError(t, err)
	requests := batch.New(batch.Config[schema.ApiRequest]{
		Name:          "analytics_compression",
		BatchSize:     1,
		BufferSize:    32,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		Flush:         func(context.Context, []schema.ApiRequest) {},
	})
	t.Cleanup(requests.Close)
	server, err := zen.New(zen.Config{})
	require.NoError(t, err)
	routes.Register(server, &routes.Services{
		Validator:                  validator,
		ApiRequests:                requests,
		AnalyticsConnectionManager: manager,
		Auth: &analyticsAuth{principal: &principal.Principal{
			Type:                  principal.TypeAPIKey,
			AuthorizedWorkspaceID: workspaceID,
			Permissions:           []string{"unkey:v1:" + workspaceID + ":**#*"},
		}},
	}, zen.InstanceInfo{})

	for _, tt := range []struct {
		path  string
		table string
	}{
		{path: "/v2/analytics.getVerifications", table: "key_verifications_v1"},
		{path: "/v2/analytics.getRatelimits", table: "ratelimits_v1"},
		{path: "/v2/analytics.getGatewayRequests", table: "gateway_requests_v1"},
		{path: "/v2/analytics.getRuntimeLogs", table: "runtime_logs_v1"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			for _, encoding := range []string{"", "gzip", "zstd"} {
				t.Run(encoding, func(t *testing.T) {
					manager.rows = []map[string]any{{"region": strings.Repeat("region-", 512)}}
					request, err := json.Marshal(openapi.V2AnalyticsGetVerificationsRequestBody{
						Query: "SELECT region FROM " + tt.table,
					})
					require.NoError(t, err)
					req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(request))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+uid.New(uid.KeyPrefix))
					req.Header.Set("Accept-Encoding", encoding)
					rec := httptest.NewRecorder()
					server.Mux().ServeHTTP(rec, req)
					require.Equal(t, http.StatusOK, rec.Code, "%s", rec.Body.Bytes())
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
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(request))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+uid.New(uid.KeyPrefix))
			req.Header.Set("Accept-Encoding", "zstd, gzip")
			rec := httptest.NewRecorder()
			server.Mux().ServeHTTP(rec, req)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "%s", rec.Body.Bytes())
			require.Empty(t, rec.Header().Get("Content-Encoding"))
		})
	}

	t.Run("other routes stay unchanged", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/liveness", nil)
		req.Header.Set("Accept-Encoding", "zstd, gzip")
		rec := httptest.NewRecorder()
		server.Mux().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get("Content-Encoding"))
		require.Empty(t, rec.Header().Get("Vary"))
	})
}

type analyticsAuth struct {
	principal *principal.Principal
}

func (a *analyticsAuth) Authenticate(_ context.Context, s *zen.Session) (*principal.Principal, error) {
	s.SetPrincipal(a.principal)
	return a.principal, nil
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

func decode(t *testing.T, rec *httptest.ResponseRecorder) []byte {
	t.Helper()
	switch rec.Header().Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, reader.Close()) })
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		return body
	case "zstd":
		reader, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
		require.NoError(t, err)
		t.Cleanup(reader.Close)
		body, err := reader.DecodeAll(rec.Body.Bytes(), nil)
		require.NoError(t, err)
		return body
	default:
		return rec.Body.Bytes()
	}
}
