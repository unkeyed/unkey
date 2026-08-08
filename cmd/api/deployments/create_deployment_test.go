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
		want       openapi.V2DeploymentsCreateDeploymentRequestBody
	}{{"git", `deployments create-deployment --project=p --app=a --environment=e --git={"branch":"main"}`, openapi.V2DeploymentsCreateDeploymentRequestBody{Project: "p", App: "a", Environment: "e", Git: &openapi.DeploymentSourceGit{Branch: func() *string { v := "main"; return &v }()}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStatus[openapi.V2DeploymentsCreateDeploymentRequestBody](t, Cmd(), tt.args, http.StatusCreated)
			require.Equal(t, tt.want.Project, got.Project)
			require.Equal(t, tt.want.App, got.App)
			require.Equal(t, tt.want.Environment, got.Environment)
			require.Equal(t, tt.want.Git, got.Git)
			require.Nil(t, got.Image)
			require.Nil(t, got.Deployment)
		})
	}
}

func TestCreateDeploymentRejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "missing source", args: "", want: "exactly one of"},
		{name: "null source", args: " --git=null", want: "must be a JSON object"},
		{name: "multiple sources", args: ` --git={} --image={"dockerImage":"ghcr.io/unkeyed/app:latest"}`, want: "exactly one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := "unkey deployments create-deployment --project=p --app=a --environment=e --root-key=test" + tt.args
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			err := root.Run(context.Background(), strings.Fields(args))
			require.ErrorContains(t, err, tt.want)
		})
	}
}
