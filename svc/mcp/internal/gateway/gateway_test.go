package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

const (
	publicURL = "https://mcp.unkey.com"
	issuer    = "https://api.unkey.com"
	version   = "v1.2.3"
)

// TestUnauthenticatedRequestsGetChallenge guarantees that every MCP URL,
// canonical or Speakeasy-era, answers a request it cannot serve with 401 and
// a pointer to its own protected resource metadata. That header is how an
// OAuth client discovers where to sign in, and invalid_token is what makes
// it refresh instead of starting over.
func TestUnauthenticatedRequestsGetChallenge(t *testing.T) {
	h := newHarness(t)
	cliToken, _ := h.sessionToken(t, h.clock.Now().Add(time.Hour), clisession.Audience)
	expiredToken, _ := h.sessionToken(t, h.clock.Now().Add(10*time.Second), clisession.Audience, publicURL)

	mounts := map[string]string{
		"/":            "/.well-known/oauth-protected-resource",
		"/compute":     "/.well-known/oauth-protected-resource/compute",
		"/compute/":    "/.well-known/oauth-protected-resource/compute",
		"/mcp/v2":      "/.well-known/oauth-protected-resource/mcp/v2",
		"/mcp/compute": "/.well-known/oauth-protected-resource/mcp/compute",
	}
	for path, metadataPath := range mounts {
		t.Run(path, func(t *testing.T) {
			res := h.postInitialize(t, path, http.Header{})
			require.Equal(t, http.StatusUnauthorized, res.StatusCode)
			require.Equal(t, `Bearer resource_metadata="`+publicURL+metadataPath+`"`, res.Header.Get("WWW-Authenticate"))

			for _, token := range []string{cliToken, expiredToken} {
				res = h.postInitialize(t, path, http.Header{"Authorization": {"Bearer " + token}})
				require.Equal(t, http.StatusUnauthorized, res.StatusCode)
				require.Equal(t, `Bearer resource_metadata="`+publicURL+metadataPath+`", error="invalid_token"`, res.Header.Get("WWW-Authenticate"))
			}
		})
	}
	require.Empty(t, h.api.recorded(), "rejected requests never reach the API")
}

func TestProtectedResourceMetadata(t *testing.T) {
	h := newHarness(t)
	cases := map[string]oauthex.ProtectedResourceMetadata{
		"/.well-known/oauth-protected-resource": {
			Resource:              publicURL,
			ResourceName:          "Unkey",
			ResourceDocumentation: publicURL + "/install",
		},
		"/.well-known/oauth-protected-resource/mcp/v2": {
			Resource:              publicURL + "/mcp/v2",
			ResourceName:          "Unkey",
			ResourceDocumentation: publicURL + "/install",
		},
		"/.well-known/oauth-protected-resource/compute": {
			Resource:              publicURL + "/compute",
			ResourceName:          "Unkey Compute",
			ResourceDocumentation: publicURL + "/compute/install",
		},
		"/.well-known/oauth-protected-resource/mcp/compute": {
			Resource:              publicURL + "/mcp/compute",
			ResourceName:          "Unkey Compute",
			ResourceDocumentation: publicURL + "/compute/install",
		},
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			want.AuthorizationServers = []string{issuer}
			want.BearerMethodsSupported = []string{"header"}

			res := h.get(t, path, http.Header{})
			require.Equal(t, http.StatusOK, res.StatusCode)
			var got oauthex.ProtectedResourceMetadata
			require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
			require.Equal(t, want, got)
		})
	}
}

func TestInstallPagesAndRedirects(t *testing.T) {
	h := newHarness(t)

	for _, toolset := range tools.Toolsets {
		installPath := strings.TrimSuffix(toolset.Path, "/") + "/install"
		res := h.get(t, installPath, http.Header{})
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Contains(t, res.Header.Get("Content-Type"), "text/html")
		page, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		for _, tool := range h.registry.Tools(toolset.ID) {
			require.Contains(t, string(page), tool.Name)
		}
		require.Contains(t, string(page), "claude mcp add --transport http "+toolset.ServerName)

		res = h.get(t, toolset.LegacyPath+"/install", http.Header{})
		require.Equal(t, http.StatusPermanentRedirect, res.StatusCode)
		require.Equal(t, installPath, res.Header.Get("Location"))

		for _, mountPath := range []string{toolset.Path, toolset.LegacyPath} {
			res = h.get(t, mountPath, http.Header{"Accept": {"text/html,application/xhtml+xml"}})
			require.Equal(t, http.StatusFound, res.StatusCode, mountPath)
			require.Equal(t, installPath, res.Header.Get("Location"), mountPath)
		}
	}

	res := h.get(t, "/does-not-exist", http.Header{})
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

// TestToolsAreServedOnEveryMountWithEveryCredentialHeader guarantees that
// Speakeasy-era client configs keep working: legacy URLs serve the same
// tools as the canonical ones, and both legacy root key headers are
// forwarded to the API as a Bearer token.
func TestToolsAreServedOnEveryMountWithEveryCredentialHeader(t *testing.T) {
	h := newHarness(t)
	rootKey := uid.New(uid.KeyPrefix)
	cases := []struct {
		path    string
		toolset tools.ToolsetID
		header  http.Header
	}{
		{path: "/", toolset: tools.ToolsetV2, header: http.Header{"Authorization": {"Bearer " + rootKey}}},
		{path: "/mcp/v2", toolset: tools.ToolsetV2, header: http.Header{"Mcp-Unkey-V2-Root-Key": {rootKey}}},
		{path: "/compute", toolset: tools.ToolsetCompute, header: http.Header{"Mcp-Unkey-V2-Bearer": {rootKey}}},
		{path: "/mcp/compute", toolset: tools.ToolsetCompute, header: http.Header{"Mcp-Unkey-V2-Bearer": {"Bearer " + rootKey}}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			session := h.connect(t, tc.path, tc.header)
			listed, err := session.ListTools(context.Background(), nil)
			require.NoError(t, err)
			var names []string
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)
			}
			var want []string
			for _, tool := range h.registry.Tools(tc.toolset) {
				want = append(want, tool.Name)
			}
			require.ElementsMatch(t, want, names)

			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "unkey_v2_workspace_get_limits"})
			require.NoError(t, err)
			require.False(t, result.IsError)
			request := h.api.last(t)
			require.Equal(t, "/v2/workspace.getLimits", request.path)
			require.Equal(t, "Bearer "+rootKey, request.header.Get("Authorization"))
		})
	}
}

func TestToolCallForwarding(t *testing.T) {
	h := newHarness(t)
	token, _ := h.sessionToken(t, h.clock.Now().Add(15*time.Minute), clisession.Audience, publicURL+"/compute")
	session := h.connect(t, "/", http.Header{"Authorization": {"Bearer " + token}})
	call := func(t *testing.T, name string, arguments any) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
		require.NoError(t, err)
		return result
	}

	t.Run("arguments are the request body", func(t *testing.T) {
		requestID := uid.New(uid.RequestPrefix)
		response := openapi.V2ApisGetApiResponseBody{
			Meta: openapi.Meta{RequestId: requestID},
			Data: openapi.V2ApisGetApiResponseData{Id: uid.New(uid.APIPrefix), Name: "payments"},
		}
		h.api.respond(http.StatusOK, response)
		arguments := openapi.V2ApisGetApiRequestBody{ApiId: response.Data.Id}

		result := call(t, "unkey_v2_apis_get_api", arguments)

		require.False(t, result.IsError)
		require.JSONEq(t, string(mustJSON(t, response)), textOf(t, result))
		request := h.api.last(t)
		require.Equal(t, "/v2/apis.getApi", request.path)
		require.Equal(t, "Bearer "+token, request.header.Get("Authorization"))
		require.Equal(t, "unkey-mcp/"+version, request.header.Get("User-Agent"))
		require.Equal(t, "application/json", request.header.Get("Content-Type"))
		require.JSONEq(t, string(mustJSON(t, arguments)), string(request.body))
	})

	t.Run("missing arguments send an empty object", func(t *testing.T) {
		h.api.respond(http.StatusOK, openapi.V2ApisGetApiResponseBody{Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)}})
		call(t, "unkey_v2_workspace_get_flags", nil)
		require.Equal(t, "{}", string(h.api.last(t).body))
	})

	t.Run("non-object bodies are unwrapped", func(t *testing.T) {
		h.api.respond(http.StatusOK, openapi.V2ApisGetApiResponseBody{Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)}})
		checks := openapi.V2RatelimitMultiLimitRequestBody{{Identifier: uid.New(uid.TestPrefix), Limit: 10, Duration: 60_000}}
		call(t, "unkey_v2_ratelimit_multi_limit", map[string]any{"body": checks})
		require.JSONEq(t, string(mustJSON(t, checks)), string(h.api.last(t).body))
	})

	t.Run("API errors are tool errors with the API envelope", func(t *testing.T) {
		response := openapi.BadRequestErrorResponse{
			Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)},
			Error: openapi.BadRequestErrorDetails{
				Title:  "Bad Request",
				Detail: "apiId is required",
				Status: http.StatusBadRequest,
				Type:   "https://unkey.com/docs/errors/unkey/application/invalid_input",
			},
		}
		h.api.respond(http.StatusBadRequest, response)

		result := call(t, "unkey_v2_apis_get_api", openapi.V2ApisGetApiRequestBody{})

		require.True(t, result.IsError)
		require.JSONEq(t, string(mustJSON(t, response)), textOf(t, result))
	})

	t.Run("arguments that are not an object are not forwarded", func(t *testing.T) {
		before := len(h.api.recorded())
		result := call(t, "unkey_v2_ratelimit_multi_limit", map[string]any{"checks": []string{}})
		require.True(t, result.IsError)
		require.Len(t, h.api.recorded(), before)
	})

	t.Run("unreachable API is a tool error", func(t *testing.T) {
		h.api.server.Close()
		result := call(t, "unkey_v2_workspace_get_flags", nil)
		require.True(t, result.IsError)
		require.Contains(t, textOf(t, result), "The Unkey API call failed")
	})
}

// TestToolCallAnalytics guarantees one analytics row per tools/call that
// attributes the call without storing its arguments or result, and that an
// unverified workspace claim is recorded only after the API accepted the
// token.
func TestToolCallAnalytics(t *testing.T) {
	h := newHarness(t)
	token, claims := h.sessionToken(t, h.clock.Now().Add(15*time.Minute), clisession.Audience, publicURL)
	oauth := h.connect(t, "/mcp/v2", http.Header{"Authorization": {"Bearer " + token}})
	call := func(t *testing.T, session *mcp.ClientSession, name string, arguments any) schema.McpToolCallV1 {
		t.Helper()
		before := len(h.calls.recorded())
		_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
		require.NoError(t, err)
		rows := h.calls.recorded()
		require.Len(t, rows, before+1)
		return rows[len(rows)-1]
	}

	t.Run("accepted OAuth call", func(t *testing.T) {
		requestID := uid.New(uid.RequestPrefix)
		h.api.respond(http.StatusOK, openapi.V2ApisGetApiResponseBody{Meta: openapi.Meta{RequestId: requestID}, Data: openapi.V2ApisGetApiResponseData{Id: uid.New(uid.APIPrefix), Name: "payments"}})

		row := call(t, oauth, "unkey_v2_apis_get_api", openapi.V2ApisGetApiRequestBody{ApiId: uid.New(uid.APIPrefix)})

		require.NotEmpty(t, row.RequestID)
		require.Equal(t, h.clock.Now().UnixMilli(), row.Time)
		require.Equal(t, schema.McpToolCallV1{
			RequestID:     row.RequestID,
			Time:          row.Time,
			Toolset:       "v2",
			Mount:         "/mcp/v2",
			Tool:          "unkey_v2_apis_get_api",
			Outcome:       "ok",
			HttpStatus:    http.StatusOK,
			ErrorType:     "",
			LatencyMs:     0,
			AuthKind:      "oauth",
			WorkspaceID:   claims.WorkspaceID,
			SessionID:     claims.SessionID,
			ApiRequestID:  requestID,
			ClientName:    "gateway-test",
			ClientVersion: "1.0.0",
		}, row)
	})

	t.Run("rejected OAuth call does not trust the workspace claim", func(t *testing.T) {
		errorType := "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions"
		h.api.respond(http.StatusForbidden, openapi.ForbiddenErrorResponse{
			Meta:  openapi.Meta{RequestId: uid.New(uid.RequestPrefix)},
			Error: openapi.BaseError{Title: "Forbidden", Detail: "missing permission", Status: http.StatusForbidden, Type: errorType},
		})

		row := call(t, oauth, "unkey_v2_apis_get_api", openapi.V2ApisGetApiRequestBody{ApiId: uid.New(uid.APIPrefix)})

		require.Equal(t, "api_error", row.Outcome)
		require.Equal(t, int32(http.StatusForbidden), row.HttpStatus)
		require.Equal(t, errorType, row.ErrorType)
		require.Empty(t, row.WorkspaceID)
		require.Equal(t, claims.SessionID, row.SessionID)
	})

	t.Run("root key call", func(t *testing.T) {
		h.api.respond(http.StatusOK, openapi.V2ApisGetApiResponseBody{Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)}})
		session := h.connect(t, "/compute", http.Header{"Mcp-Unkey-V2-Bearer": {uid.New(uid.KeyPrefix)}})

		row := call(t, session, "unkey_v2_workspace_get_limits", nil)

		require.Equal(t, "compute", row.Toolset)
		require.Equal(t, "/compute", row.Mount)
		require.Equal(t, "root_key", row.AuthKind)
		require.Empty(t, row.WorkspaceID)
		require.Empty(t, row.SessionID)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		row := call(t, oauth, "unkey_v2_ratelimit_multi_limit", map[string]any{"checks": []string{}})
		require.Equal(t, "rejected", row.Outcome)
		require.Zero(t, row.HttpStatus)
	})

	t.Run("client named by User-Agent when clientInfo is absent", func(t *testing.T) {
		h.api.respond(http.StatusOK, openapi.V2ApisGetApiResponseBody{Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)}})
		before := len(h.calls.recorded())
		res := h.post(t, "/", http.Header{"Authorization": {"Bearer " + token}, "User-Agent": {"claude-code/2.1.4 (external, cli)"}}, rpcRequest{
			JSONRPC: "2.0",
			ID:      7,
			Method:  "tools/call",
			Params:  mcp.CallToolParams{Name: "unkey_v2_workspace_get_limits"},
		})
		require.Equal(t, http.StatusOK, res.StatusCode)
		rows := h.calls.recorded()
		require.Len(t, rows, before+1)
		require.Equal(t, "claude-code", rows[len(rows)-1].ClientName)
		require.Equal(t, "2.1.4", rows[len(rows)-1].ClientVersion)
	})

	t.Run("unreachable API", func(t *testing.T) {
		h.api.server.Close()
		row := call(t, oauth, "unkey_v2_workspace_get_limits", nil)
		require.Equal(t, "transport_error", row.Outcome)
		require.Zero(t, row.HttpStatus)
		require.Empty(t, row.WorkspaceID)
	})
}

type harness struct {
	calls    *callRecorder
	api      *fakeAPI
	server   *httptest.Server
	clock    *clock.TestClock
	registry *tools.Registry
	signer   *jwt.HS256Signer[clisession.Claims]
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	api := newFakeAPI(t)
	registry, err := tools.Load(openapi.Spec)
	require.NoError(t, err)
	signer, err := jwt.NewHS256Signer[clisession.Claims]([]byte(uid.New(uid.TestPrefix, 32)))
	require.NoError(t, err)

	h := &harness{calls: &callRecorder{mu: sync.Mutex{}, rows: nil}, api: api, clock: clock.NewTestClock(time.Now()), registry: registry, signer: signer, server: nil}
	handler, err := New(Config{
		PublicURL:  publicURL,
		APIURL:     api.server.URL,
		Issuer:     issuer,
		Version:    version,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Clock:      h.clock,
		Calls:      h.calls,
	}, registry)
	require.NoError(t, err)
	h.server = httptest.NewServer(handler)
	t.Cleanup(h.server.Close)
	return h
}

func (h *harness) sessionToken(t *testing.T, expiresAt time.Time, audience ...string) (string, clisession.Claims) {
	t.Helper()
	claims := clisession.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: clisession.Issuer, Audience: audience, ExpiresAt: expiresAt.Unix()},
		WorkspaceID:      uid.New(uid.WorkspacePrefix),
		SessionID:        uid.New(uid.CLISessionPrefix),
		TokenUse:         clisession.TokenUse,
	}
	token, err := h.signer.Sign(claims)
	require.NoError(t, err)
	return token, claims
}

func (h *harness) connect(t *testing.T, path string, header http.Header) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "gateway-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             h.server.URL + path,
		HTTPClient:           &http.Client{Transport: headerTransport{header: header}},
		DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

func (h *harness) postInitialize(t *testing.T, path string, header http.Header) *http.Response {
	t.Helper()
	return h.post(t, path, header, rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: mcp.InitializeParams{
			ProtocolVersion: "2025-06-18",
			ClientInfo:      &mcp.Implementation{Name: "gateway-test", Version: "1.0.0"},
			Capabilities:    &mcp.ClientCapabilities{},
		},
	})
}

// post sends one JSON-RPC message the way a 2025-06-18 client does, without
// per-request clientInfo.
func (h *harness) post(t *testing.T, path string, header http.Header, message rpcRequest) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.server.URL+path, bytes.NewReader(mustJSON(t, message)))
	require.NoError(t, err)
	req.Header = header.Clone()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	return h.do(t, req)
}

func (h *harness) get(t *testing.T, path string, header http.Header) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.server.URL+path, nil)
	require.NoError(t, err)
	req.Header = header.Clone()
	return h.do(t, req)
}

func (h *harness) do(t *testing.T, req *http.Request) *http.Response {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

type headerTransport struct {
	header http.Header
}

func (h headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for name, values := range h.header {
		req.Header[name] = values
	}
	return http.DefaultTransport.RoundTrip(req)
}

type callRecorder struct {
	mu   sync.Mutex
	rows []schema.McpToolCallV1
}

func (r *callRecorder) Buffer(row schema.McpToolCallV1) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
}

func (r *callRecorder) recorded() []schema.McpToolCallV1 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]schema.McpToolCallV1(nil), r.rows...)
}

type recordedRequest struct {
	path   string
	header http.Header
	body   []byte
}

// fakeAPI stands in for api.unkey.com: it records every forwarded call and
// answers with the configured status and typed response body.
type fakeAPI struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	status   int
	response []byte
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	api := &fakeAPI{server: nil, requests: nil, status: http.StatusOK, response: mustJSON(t, openapi.V2ApisGetApiResponseBody{
		Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)},
		Data: openapi.V2ApisGetApiResponseData{Id: uid.New(uid.APIPrefix), Name: "default"},
	})}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		api.mu.Lock()
		api.requests = append(api.requests, recordedRequest{path: r.URL.Path, header: r.Header.Clone(), body: body})
		status, response := api.status, api.response
		api.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(response)
	}))
	t.Cleanup(api.server.Close)
	return api
}

func (a *fakeAPI) respond(status int, response any) {
	encoded, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status, a.response = status, encoded
}

func (a *fakeAPI) recorded() []recordedRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]recordedRequest(nil), a.requests...)
}

func (a *fakeAPI) last(t *testing.T) recordedRequest {
	t.Helper()
	requests := a.recorded()
	require.NotEmpty(t, requests)
	return requests[len(requests)-1]
}

func textOf(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}
