package deployments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"net/http"
	"testing"
)

func TestPromoteDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsPromoteDeploymentRequestBody
	}{{"request", "deployments promote-deployment --deployment-id=x", openapi.V2DeploymentsPromoteDeploymentRequestBody{DeploymentId: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V2DeploymentsPromoteDeploymentRequestBody](t, Cmd(), tt.args, http.StatusAccepted)
			require.Equal(t, tt.want, got)
		})
	}
}
