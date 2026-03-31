package util

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

	"github.com/unkeyed/unkey/pkg/cli"
)

// CaptureRequest runs a CLI command against a local test server, captures the
// JSON request body, and unmarshals it into T. The test server responds with a
// minimal valid envelope so the SDK does not error.
//
// Example:
//
//	req, err := util.CaptureRequest[handler.Request](t, Cmd(), "keys create-key --api-id=api_123")
//	require.NoError(t, err)
//	require.Equal(t, handler.Request{ApiId: "api_123"}, req)
func CaptureRequest[T any](t *testing.T, cmd *cli.Command, args string) T {
	t.Helper()
	return CaptureRequestWithResponse[T](t, cmd, args, `{"meta":{"requestId":"test"},"data":{}}`)
}

// CaptureRequestWithResponse is like CaptureRequest but lets the caller supply
// a custom JSON response body. Use this when the default object-shaped data
// envelope (`"data":{}`) does not match the SDK's expected response type (e.g.
// endpoints that return an array in `data`).
func CaptureRequestWithResponse[T any](t *testing.T, cmd *cli.Command, args string, response string) T {
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
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)

	// Suppress stdout.
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

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
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
