package deployments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestGetDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsGetDeploymentRequestBody
	}{{"request", "deployments get-deployment --deployment-id=x", openapi.V2DeploymentsGetDeploymentRequestBody{DeploymentId: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2DeploymentsGetDeploymentRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}
