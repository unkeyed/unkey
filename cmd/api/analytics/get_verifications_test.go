package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// captureRequest is a local variant of util.CaptureRequest that returns a
// response with "data" as an empty array instead of an empty object.  The
// analytics SDK endpoint deserialises "data" as []json.RawMessage, so the
// generic harness response shape does not work here.
func captureRequest[T any](t *testing.T, cmd *cli.Command, args string) T {
	t.Helper()

	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body = b
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"requestId":"test"},"data":[]}`))
	}))
	t.Cleanup(srv.Close)

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fullArgs := fmt.Sprintf("unkey %s --api-url=%s --root-key=test_key", args, srv.URL)
	root := &cli.Command{
		Name:     "unkey",
		Commands: []*cli.Command{cmd},
	}

	runErr := root.Run(context.Background(), strings.Fields(fullArgs))

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = origStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if runErr != nil {
		t.Fatalf("CLI command failed: %v", runErr)
	}

	var req T
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal request body: %v\nbody: %s", err, string(body))
	}

	return req
}

func TestGetVerifications(t *testing.T) {
	// Note: the shared test harness splits args with strings.Fields, so query
	// values must not contain spaces.  This is fine because we are testing
	// flag-to-request mapping, not SQL validity.
	tests := []struct {
		name string
		args string
		want openapi.V2AnalyticsGetVerificationsRequestBody
	}{
		{
			name: "simple query",
			args: `analytics get-verifications --query=SELECT(1)`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT(1)",
			},
		},
		{
			name: "count query without spaces",
			args: `analytics get-verifications --query=SELECT+COUNT(*)+FROM+verifications`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT+COUNT(*)+FROM+verifications",
			},
		},
		{
			name: "query with filter",
			args: `analytics get-verifications --query=SELECT+key_id,outcome+FROM+key_verifications_v1+WHERE+outcome='VALID'`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT+key_id,outcome+FROM+key_verifications_v1+WHERE+outcome='VALID'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := captureRequest[openapi.V2AnalyticsGetVerificationsRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
