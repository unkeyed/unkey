package github

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
)

func TestInstallApp(t *testing.T) {
	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/github.installApp", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"meta":{"requestId":"test"},"data":{"url":"https://github.com/apps/unkey/installations/new","expiresAt":1700000900000}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
		_ = reader.Close()
		_ = writer.Close()
	})

	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	args := fmt.Sprintf("unkey github install-app --api-url=%s --root-key=test_key", server.URL)
	require.NoError(t, root.Run(context.Background(), strings.Fields(args)))
	require.NoError(t, writer.Close())
	os.Stdout = originalStdout
	_, err = io.Copy(&bytes.Buffer{}, reader)
	require.NoError(t, err)
	require.Empty(t, requestBody)
}
