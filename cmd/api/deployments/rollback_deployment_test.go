package deployments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"net/http"
	"testing"
)

func TestRollbackDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsRollbackDeploymentRequestBody
	}{{"request", "deployments rollback-deployment --deployment-id=x", openapi.V2DeploymentsRollbackDeploymentRequestBody{DeploymentId: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V2DeploymentsRollbackDeploymentRequestBody](t, Cmd(), tt.args, http.StatusAccepted)
			require.Equal(t, tt.want, got)
		})
	}
}
