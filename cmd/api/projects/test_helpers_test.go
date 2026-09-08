package projects

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

func captureAcceptedRequest[T any](t *testing.T, cmd *cli.Command, args string) T {
	t.Helper()
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var err error
		body, err = io.ReadAll(request.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"meta":{"requestId":"test"},"data":{}}`))
	}))
	t.Cleanup(server.Close)
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = original
		_ = writer.Close()
		_ = reader.Close()
	})
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{cmd}}
	runErr := root.Run(context.Background(), strings.Fields(fmt.Sprintf("unkey %s --api-url=%s --root-key=test", args, server.URL)))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	_, _ = io.Copy(&bytes.Buffer{}, reader)
	if runErr != nil {
		t.Fatalf("CLI command failed: %v", runErr)
	}
	var request T
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatal(err)
	}
	return request
}
