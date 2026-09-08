package deployments

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListDeployments(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2DeploymentsListDeploymentsRequestBody
	}{
		{"request", "deployments list-deployments ", openapi.V2DeploymentsListDeploymentsRequestBody{Project: nil, App: nil, Environment: nil, Status: nil, Limit: ptr.P(100), Cursor: nil}},
		{"valid statuses", "deployments list-deployments --status=ready,failed", openapi.V2DeploymentsListDeploymentsRequestBody{Project: nil, App: nil, Environment: nil, Status: ptr.P([]openapi.DeploymentStatus{openapi.DeploymentStatusReady, openapi.DeploymentStatusFailed}), Limit: ptr.P(100), Cursor: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2DeploymentsListDeploymentsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, got)
		})
	}
}

func TestListDeploymentsStatusValidation(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields("unkey deployments list-deployments --status=ready,not-a-status --root-key=test"))
	require.ErrorContains(t, err, `invalid status "not-a-status"`)
}
