package deployments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"net/http"
	"testing"
)

func TestStartDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsStartDeploymentRequestBody
	}{{"request", "deployments start-deployment --deployment-id=x", openapi.V2DeploymentsStartDeploymentRequestBody{DeploymentId: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V2DeploymentsStartDeploymentRequestBody](t, Cmd(), tt.args, http.StatusAccepted)
			require.Equal(t, tt.want, got)
		})
	}
}
