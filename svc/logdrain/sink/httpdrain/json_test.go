package httpdrain

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
)

func TestDeliverJSONTreatsResponseBodyAsDiagnostic(t *testing.T) {
	for _, responseBody := range []string{`{"code":9}`, "accepted"} {
		t.Run(responseBody, func(t *testing.T) {
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
				if _, err := io.WriteString(w, responseBody); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			drain := newTestSink(t, Config{
				Endpoint: server.URL,
				Format:   logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
			})
			result, err := drain.Deliver(t.Context(), testBatch())
			require.NoError(t, err)
			require.True(t, result.Acknowledged)
			require.Equal(t, http.StatusOK, result.HTTPStatus)
			require.Equal(t, responseBody, result.ResponseBody)
			require.Equal(t, int64(len(receivedBody)), result.RequestBodyBytes)
			require.Equal(t, "application/json", contentType)
			var records []struct {
				ID string `json:"id"`
			}
			require.NoError(t, json.Unmarshal(receivedBody, &records))
			require.Len(t, records, 2)
			require.Equal(t, "evt_1", records[0].ID)
			require.Equal(t, "evt_2", records[1].ID)
		})
	}
}
