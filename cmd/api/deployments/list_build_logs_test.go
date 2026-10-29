package deployments

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListBuildLogs(t *testing.T) {
	deploymentID := uid.New(uid.DeploymentPrefix)
	stepID := uid.New(uid.TestPrefix)
	args := "deployments list-build-logs --deployment-id=" + deploymentID
	tests := []struct {
		name string
		args string
		want openapi.V2DeploymentsListBuildLogsRequestBody
	}{
		{
			name: "only required flags uses SDK page size default",
			args: args,
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				Limit:        new(100),
			},
		},
		{
			name: "filter by build step",
			args: args + " --step-id=" + stepID,
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				StepId:       new(stepID),
				Limit:        new(100),
			},
		},
		{
			name: "continue after cursor",
			args: args + " --cursor=1704067201000001",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				Cursor:       new("1704067201000001"),
				Limit:        new(100),
			},
		},
		{
			name: "custom page size",
			args: args + " --limit=50",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				Limit:        new(50),
			},
		},
		{
			name: "explicit zero is preserved for API validation",
			args: args + " --limit=0",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				Limit:        new(0),
			},
		},
		{
			name: "explicit empty step is preserved for API validation",
			args: args + " --step-id=",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				StepId:       new(""),
				Limit:        new(100),
			},
		},
		{
			name: "explicit empty cursor is preserved for API validation",
			args: args + " --cursor=",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				Cursor:       new(""),
				Limit:        new(100),
			},
		},
		{
			name: "all flags",
			args: args + " --step-id=" + stepID + " --cursor=1704067201000001 --limit=50",
			want: openapi.V2DeploymentsListBuildLogsRequestBody{
				DeploymentId: deploymentID,
				StepId:       new(stepID),
				Cursor:       new("1704067201000001"),
				Limit:        new(50),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2DeploymentsListBuildLogsRequestBody](t, Cmd(), tt.args, []openapi.BuildLogEntry{})
			require.Equal(t, tt.want, got)
		})
	}
}
