package deployments

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

func captureStatus[T any](t *testing.T, cmd *cli.Command, args string, status int) T {
	t.Helper()
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		body, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, err = w.Write([]byte(`{"meta":{"requestId":"test"},"data":{}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = stdout; _ = reader.Close() })
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{cmd}}
	err = root.Run(context.Background(), strings.Fields(fmt.Sprintf("unkey %s --api-url=%s --root-key=test", args, srv.URL)))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	_, _ = io.Copy(io.Discard, bytes.NewBuffer(nil))
	var got T
	require.NoError(t, json.Unmarshal(body, &got))
	return got
}

func TestStopDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsStopDeploymentRequestBody
	}{{"request", "deployments stop-deployment --deployment-id=x", openapi.V2DeploymentsStopDeploymentRequestBody{DeploymentId: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V2DeploymentsStopDeploymentRequestBody](t, Cmd(), tt.args, http.StatusAccepted)
			require.Equal(t, tt.want, got)
		})
	}
}
