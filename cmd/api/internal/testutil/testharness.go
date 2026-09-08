// Package testutil provides shared test helpers for CLI API commands.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/shlex"
	"github.com/unkeyed/unkey/pkg/cli"
)

type responseMeta struct {
	RequestID string `json:"requestId"`
}

type responseEnvelope struct {
	Meta responseMeta `json:"meta"`
	Data any          `json:"data"`
}

// CaptureRequest runs a CLI command against a local test server, captures the
// JSON request body, and unmarshals it into T. The test server responds with a
// minimal valid envelope so the SDK does not error.
//
// Example:
//
//	req := testutil.CaptureRequest[handler.Request](t, Cmd(), "keys create-key --api-id=api_123")
//	require.Equal(t, handler.Request{ApiId: "api_123"}, req)
func CaptureRequest[T any](t *testing.T, cmd *cli.Command, args string) T {
	t.Helper()
	return CaptureRequestWithData[T](t, cmd, args, struct{}{})
}

// CaptureRequestWithData is like CaptureRequest but lets the caller supply the
// response envelope's data value. Use this for endpoints that return an array.
func CaptureRequestWithData[T any](t *testing.T, cmd *cli.Command, args string, responseData any) T {
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
		response := responseEnvelope{
			Meta: responseMeta{RequestID: "test"},
			Data: responseData,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	// Suppress stdout.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = origStdout
		_ = w.Close()
		_ = r.Close()
	})

	fullArgs := fmt.Sprintf("unkey %s --api-url=%s --root-key=test_key", args, srv.URL)
	root := &cli.Command{
		Name:     "unkey",
		Commands: []*cli.Command{cmd},
	}

	parsedArgs, err := shlex.Split(fullArgs)
	if err != nil {
		t.Fatalf("failed to parse command arguments: %v", err)
	}
	runErr := root.Run(context.Background(), parsedArgs)

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
