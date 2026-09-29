package httpdrain

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func TestResponseSizeLimit(t *testing.T) {
	for _, format := range []logdrainv1.HttpBodyFormat{
		logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC,
		logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
		logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON,
	} {
		for name, body := range map[string]string{
			"rejection beyond prefix": strings.Repeat(" ", 4096) + `{"code":9}`,
			"oversized success":       `{"code":0}` + strings.Repeat(" ", 4088),
			"below limit":             strings.Repeat(" ", 4086) + `{"code":0}`,
			"exact limit":             strings.Repeat(" ", 4087) + `{"code":0}`,
		} {
			t.Run(format.String()+"/"+name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Retry-After", "17")
					_, err := io.WriteString(w, body)
					if err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				drain := newTestSink(t, Config{
					Endpoint: server.URL,
					Format:   format,
				})
				result, err := drain.Deliver(t.Context(), testBatch())
				if format == logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC && len(body) > 4096 {
					require.ErrorContains(t, err, "HEC response exceeds")
					require.False(t, result.Acknowledged)
				} else {
					require.NoError(t, err)
					require.True(t, result.Acknowledged)
				}
				require.Equal(t, http.StatusOK, result.HTTPStatus)
				require.Equal(t, 17*time.Second, result.RetryAfter)
				require.Positive(t, result.RequestBodyBytes)
				require.Equal(t, strings.TrimSpace(body[:min(len(body), 4096)]), result.ResponseBody)
			})
		}
	}
}

func TestDeliverHECDoesNotSendPartiallyEncodedBatch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	drain := newTestSink(t, Config{
		Endpoint: server.URL,
		Format:   logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC,
	})
	batch := testBatch()
	batch.Events[1].Payload = sink.RuntimeLogPayload{
		LogID:      "invalid",
		Attributes: json.RawMessage(`{"unfinished":`),
	}
	result, err := drain.Deliver(t.Context(), batch)
	require.ErrorContains(t, err, "marshal batch")
	require.Equal(t, sink.Result{}, result)
	require.Zero(t, requests.Load())
}
