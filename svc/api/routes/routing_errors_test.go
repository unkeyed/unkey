package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes"
)

func TestRoutingErrors(t *testing.T) {
	logs := loggertest.Install(t)
	validator, err := validation.New()
	require.NoError(t, err)
	srv, err := zen.New(zen.Config{MaxRequestBodySize: 16})
	require.NoError(t, err)
	routes.Register(srv, &routes.Services{Validator: validator}, zen.InstanceInfo{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-serveErr)
	})
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tt := range []struct {
		name   string
		method string
		path   string
		accept string
		status int
		allow  string
		detail string
	}{
		{"root", http.MethodGet, "/", "application/json", 404, "", "The requested endpoint does not exist."},
		{"typo", http.MethodPost, "/v2/keys.verfyKey", "application/json", 404, "", "The requested endpoint does not exist."},
		{"unknown version", http.MethodGet, "/v99/keys.verifyKey", "application/json", 404, "", "The requested endpoint does not exist."},
		{"trailing slash", http.MethodGet, "/v2/liveness/", "application/json", 404, "", "The requested endpoint does not exist."},
		{"problem json", http.MethodPost, "/v2/unknown", "application/problem+json", 404, "", "The requested endpoint does not exist."},
		{"wrong get", http.MethodGet, "/v2/keys.verifyKey", "application/json", 405, "POST", "The request method is not supported for this endpoint."},
		{"wrong post", http.MethodPost, "/v2/liveness", "application/json", 405, "GET, HEAD", "The request method is not supported for this endpoint."},
		{"wrong options", http.MethodOptions, "/v2/keys.verifyKey", "application/problem+json", 405, "POST", "The request method is not supported for this endpoint."},
		{"head not found", http.MethodHead, "/unknown", "application/json", 404, "", ""},
		{"head wrong method", http.MethodHead, "/v2/keys.verifyKey", "application/json", 405, "POST", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tt.method, "http://"+listener.Addr().String()+tt.path, strings.NewReader("invalid body exceeding the size limit"))
			require.NoError(t, err)
			req.Header.Set("Accept", tt.accept)
			snapshot := logs.Snapshot()
			res, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			for _, record := range logs.Since(snapshot) {
				attrs := loggertest.FlatAttrs(record)
				require.Empty(t, attrs["http.request_id"])
				require.Empty(t, attrs["requestId"])
			}
			require.Equal(t, tt.status, res.StatusCode, "%s", body)
			require.Equal(t, tt.allow, res.Header.Get("Allow"))
			require.Equal(t, []string{tt.accept}, res.Header.Values("Content-Type"))
			if tt.method == http.MethodHead {
				require.Empty(t, body)
				return
			}

			var response struct {
				Meta  openapi.Meta      `json:"meta"`
				Error openapi.BaseError `json:"error"`
			}
			require.NoError(t, json.Unmarshal(body, &response))
			require.NotEmpty(t, response.Meta.RequestId)
			require.Equal(t, openapi.BaseError{
				Status: tt.status,
				Title:  http.StatusText(tt.status),
				Type:   "about:blank",
				Detail: tt.detail,
			}, response.Error)
		})
	}

	t.Run("registered route", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String()+"/v2/liveness", nil)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
		require.Equal(t, http.StatusOK, res.StatusCode)
		var response openapi.V2LivenessResponseBody
		require.NoError(t, json.NewDecoder(res.Body).Decode(&response))
		require.Equal(t, "we're cooking", response.Data.Message)
		require.NotEmpty(t, response.Meta.RequestId)
	})
}

func TestRegisteredRequestLogging(t *testing.T) {
	logs := loggertest.Install(t)
	for _, tc := range []struct {
		name      string
		code      codes.URN
		status    int
		event     string
		bodyLimit int64
		closeErr  bool
	}{
		{name: "invalid_input", code: codes.App.Validation.InvalidInput.URN(), status: 400},
		{name: "body_too_large", status: 400, bodyLimit: 1},
		{name: "body_close_failure", status: 400, closeErr: true},
		{name: "invalid_credentials", code: codes.Auth.Authentication.CredentialRejected.URN(), status: 401},
		{name: "database", code: codes.App.Internal.UnexpectedError.URN(), status: 500, event: "api.request_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rows []schema.ApiRequest
			requests := batch.New(batch.Config[schema.ApiRequest]{
				Name:          "test",
				BatchSize:     10,
				BufferSize:    10,
				Consumers:     1,
				FlushInterval: time.Hour,
				Flush:         func(_ context.Context, batch []schema.ApiRequest) { rows = append(rows, batch...) },
			})
			t.Cleanup(requests.Close)
			validator, err := validation.New()
			require.NoError(t, err)
			srv, err := zen.New(zen.Config{MaxRequestBodySize: tc.bodyLimit})
			require.NoError(t, err)
			routes.Register(srv, &routes.Services{
				Validator:   validator,
				ApiRequests: requests,
				Auth:        failingAuth{err: fault.New("request failed", fault.Code(tc.code))},
			}, zen.InstanceInfo{})
			var body bytes.Buffer
			require.NoError(t, json.NewEncoder(&body).Encode(openapi.V2KeysGetKeyRequestBody{KeyId: uid.New(uid.KeyPrefix)}))
			req := httptest.NewRequest(http.MethodPost, "/v2/keys.getKey", &body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer synthetic-secret")
			if tc.closeErr {
				req.Body = bodyCloseError{Reader: req.Body}
			}
			rec := httptest.NewRecorder()
			snapshot := logs.Snapshot()
			srv.ServeHTTP(rec, req)
			requests.Close()
			require.Equal(t, tc.status, rec.Code)
			require.Len(t, rows, 1)
			require.EqualValues(t, tc.status, rows[0].ResponseStatus)
			require.NotEmpty(t, rows[0].RequestID)
			require.Contains(t, rows[0].RequestHeaders, "Authorization: [REDACTED]")
			records := logs.Since(snapshot)
			if tc.closeErr {
				require.Len(t, records, 1)
				require.Equal(t, "failed to init session", records[0].Message)
				require.Equal(t, rows[0].RequestID, loggertest.FlatAttrs(records[0])["requestId"])
				return
			}
			if tc.event == "" {
				require.Empty(t, records)
				return
			}
			require.Len(t, records, 1)
			attrs := loggertest.FlatAttrs(records[0])
			require.Equal(t, tc.event, attrs["event"])
			require.Equal(t, rows[0].RequestID, attrs["requestId"])
		})
	}
}

type failingAuth struct {
	err error
}

func (a failingAuth) Authenticate(context.Context, *zen.Session) (*principal.Principal, error) {
	return nil, a.err
}

type bodyCloseError struct {
	io.Reader
}

func (bodyCloseError) Close() error { return errors.New("body close failed") }
