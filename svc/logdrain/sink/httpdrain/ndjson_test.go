package httpdrain

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
)

func TestDeliverNDJSONReportsRejectionAndRetry(t *testing.T) {
	var receivedBody []byte
	var contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedBody = body
		contentType = r.Header.Get("Content-Type")
		w.Header().Set("Retry-After", "23")
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := io.WriteString(w, `{"code":0}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	drain := newTestSink(t, Config{
		Endpoint: server.URL,
		Format:   logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON,
	})
	result, err := drain.Deliver(t.Context(), testBatch())
	require.NoError(t, err)
	require.False(t, result.Acknowledged)
	require.Equal(t, http.StatusTooManyRequests, result.HTTPStatus)
	require.Equal(t, 23*time.Second, result.RetryAfter)
	require.Equal(t, `{"code":0}`, result.ResponseBody)
	require.Equal(t, int64(len(receivedBody)), result.RequestBodyBytes)
	require.Equal(t, "application/x-ndjson", contentType)
	lines := strings.Split(strings.TrimSuffix(string(receivedBody), "\n"), "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], `"id":"evt_1"`)
	require.Contains(t, lines[1], `"id":"evt_2"`)
}
