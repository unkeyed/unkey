package logdrain

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

const testBearer = "test-bearer"

// newTestService allows private, plain-http endpoints so the probe can reach
// httptest servers on loopback. newGuardedService keeps the guard on, as
// production does.
func newTestService() *Service {
	return New(Config{Bearer: testBearer, UnsafeAllowPrivateEndpoints: true})
}

func newGuardedService() *Service {
	return New(Config{Bearer: testBearer})
}

func newRequest(msg *ctrlv1.VerifyLogdrainEndpointRequest) *connect.Request[ctrlv1.VerifyLogdrainEndpointRequest] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+testBearer)
	return req
}

func verifyEndpoint(t *testing.T, svc *Service, url string) *ctrlv1.VerifyLogdrainEndpointResponse {
	t.Helper()
	resp, err := svc.VerifyLogdrainEndpoint(context.Background(), newRequest(&ctrlv1.VerifyLogdrainEndpointRequest{
		Url: url,
	}))
	require.NoError(t, err)
	return resp.Msg
}

func TestVerifyLogdrainEndpointAcceptsTwoHundred(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	msg := verifyEndpoint(t, newTestService(), server.URL)
	require.True(t, msg.GetOk())
	require.Equal(t, int32(http.StatusOK), msg.GetResponseStatus())
	require.Empty(t, msg.GetError())
}

func TestVerifyLogdrainEndpointReportsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("method not allowed"))
	}))
	t.Cleanup(server.Close)

	msg := verifyEndpoint(t, newTestService(), server.URL)
	require.False(t, msg.GetOk())
	require.Equal(t, int32(http.StatusMethodNotAllowed), msg.GetResponseStatus())
	require.Equal(t, "method not allowed", msg.GetResponseBody())
	require.Empty(t, msg.GetError())
}

func TestVerifyLogdrainEndpointPostsAnEmptyBatch(t *testing.T) {
	var method, contentType, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		contentType = r.Header.Get("Content-Type")
		read, _ := io.ReadAll(r.Body)
		body = string(read)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	verifyEndpoint(t, newTestService(), server.URL)
	require.Equal(t, http.MethodPost, method)
	require.Equal(t, "application/json", contentType)
	require.Equal(t, "[]", body)
}

func TestVerifyLogdrainEndpointPostsNothingForNDJSON(t *testing.T) {
	var contentType, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		read, _ := io.ReadAll(r.Body)
		body = string(read)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	_, err := newTestService().VerifyLogdrainEndpoint(context.Background(), newRequest(&ctrlv1.VerifyLogdrainEndpointRequest{
		Url:    server.URL,
		Format: ctrlv1.LogdrainBodyFormat_LOGDRAIN_BODY_FORMAT_NDJSON,
	}))
	require.NoError(t, err)
	require.Equal(t, "application/x-ndjson", contentType)
	require.Empty(t, body)
}

func TestVerifyLogdrainEndpointSendsConfiguredHeaders(t *testing.T) {
	var authorization, contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	svc := newTestService()
	resp, err := svc.VerifyLogdrainEndpoint(context.Background(), newRequest(&ctrlv1.VerifyLogdrainEndpointRequest{
		Url:     server.URL,
		Format:  ctrlv1.LogdrainBodyFormat_LOGDRAIN_BODY_FORMAT_NDJSON,
		Headers: map[string]string{"Authorization": "Bearer hunter2"},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetOk())
	require.Equal(t, "Bearer hunter2", authorization)
	require.Equal(t, "application/x-ndjson", contentType)
}

func TestVerifyLogdrainEndpointReportsUnreachableHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := server.URL
	server.Close()

	msg := verifyEndpoint(t, newTestService(), url)
	require.False(t, msg.GetOk())
	require.Zero(t, msg.GetResponseStatus())
	require.NotEmpty(t, msg.GetError())
}

// The SSRF guard blocks private addresses in the dialer, not in endpoint
// validation, so a loopback endpoint is a well-formed request that cannot
// connect. It reads as a failed test rather than an invalid one, which is what
// the caller should show.
func TestVerifyLogdrainEndpointReportsBlockedAddress(t *testing.T) {
	msg := verifyEndpoint(t, newGuardedService(), "https://127.0.0.1/ingest")
	require.False(t, msg.GetOk())
	require.Zero(t, msg.GetResponseStatus())
	require.Contains(t, msg.GetError(), "endpoint host resolved only to forbidden IP addresses")
}

func TestVerifyLogdrainEndpointRequiresABearerToken(t *testing.T) {
	_, err := newTestService().VerifyLogdrainEndpoint(
		context.Background(),
		connect.NewRequest(&ctrlv1.VerifyLogdrainEndpointRequest{Url: "https://example.com/ingest"}),
	)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// A malformed endpoint is refused outright rather than reported as a failed
// test. No change to the endpoint's own behavior would make it deliverable.
// Shape only. The dialer above catches a forbidden address.
func TestVerifyLogdrainEndpointRejectsForbiddenEndpoints(t *testing.T) {
	svc := newGuardedService()

	for name, url := range map[string]string{
		"plain http":  "http://example.com/ingest",
		"credentials": "https://user:pass@example.com/ingest",
		"relative":    "/ingest",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.VerifyLogdrainEndpoint(context.Background(), newRequest(&ctrlv1.VerifyLogdrainEndpointRequest{
				Url: url,
			}))
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}
