package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCancelDeployment(t *testing.T) {
	want := openapi.V2DeploymentsCancelDeploymentRequestBody{
		DeploymentId: uid.New(uid.DeploymentPrefix),
	}
	body, err := json.Marshal(want)
	require.NoError(t, err)
	tests := []struct {
		name string
		args string
	}{
		{
			name: "cancel by ID without confirmation",
			args: "--deployment-id=" + want.DeploymentId,
		},
		{
			name: "cancel body without confirmation",
			args: "--body=" + string(body),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := make(chan openapi.V2DeploymentsCancelDeploymentRequestBody, 1)
			response := openapi.V2DeploymentsCancelDeploymentResponseBody{
				Meta: openapi.Meta{RequestId: uid.New(uid.RequestPrefix)},
				Data: openapi.EmptyResponse{},
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v2/deployments.cancelDeployment" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var request openapi.V2DeploymentsCancelDeploymentRequestBody
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- request
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			t.Cleanup(server.Close)

			root := &cli.Command{
				Name: "unkey",
				Commands: []*cli.Command{
					Cmd(),
				},
			}
			err := root.Run(context.Background(), []string{
				"unkey",
				"deployments",
				"cancel-deployment",
				tt.args,
				"--api-url=" + server.URL,
				"--root-key=" + uid.New(uid.KeyPrefix),
			})
			require.NoError(t, err)
			select {
			case got := <-requests:
				require.Equal(t, want, got)
			default:
				t.Fatal("cancellation request was not sent")
			}
		})
	}
}
