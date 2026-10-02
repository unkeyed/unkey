package deployments

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateDeployment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V3DeploymentsCreateDeploymentRequestBody
	}{
		{
			name: "app default source",
			args: "deployments create-deployment --project=payments --app=payments-api --environment=production",
			want: openapi.V3DeploymentsCreateDeploymentRequestBody{
				Project:     "payments",
				App:         "payments-api",
				Environment: "production",
			},
		},
		{
			name: "OCI source",
			args: `deployments create-deployment --project=payments --app=payments-api --environment=production --oci={"image":"ghcr.io/acme/payments:v1.2.3"}`,
			want: openapi.V3DeploymentsCreateDeploymentRequestBody{
				Project:     "payments",
				App:         "payments-api",
				Environment: "production",
				Oci:         &openapi.DeploymentSourceOCI{Image: "ghcr.io/acme/payments:v1.2.3"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V3DeploymentsCreateDeploymentRequestBody](t, Cmd(), tt.args, http.StatusCreated)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCreateDeploymentRejectsMultipleSources(t *testing.T) {
	root := &cli.Command{
		Name:     "unkey",
		Commands: []*cli.Command{Cmd()},
	}
	err := root.Run(context.Background(), strings.Fields(`unkey deployments create-deployment --root-key=test --project=payments --app=payments-api --environment=production --git={} --oci={"image":"ghcr.io/acme/payments:v1.2.3"}`))
	require.ErrorContains(t, err, "mutually exclusive")
}
