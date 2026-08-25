package environments

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestRemoveEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsRemoveEnvironmentVariablesRequestBody
	}{{"request", "environments remove-environment-variables --project=x --app=x --environment=x --variables=A,B", openapi.V2EnvironmentsRemoveEnvironmentVariablesRequestBody{Project: "x", App: "x", Environment: "x", Variables: []string{"A", "B"}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2EnvironmentsRemoveEnvironmentVariablesRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveEnvironmentVariablesRequiresVariables(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields("unkey environments remove-environment-variables --project=p --app=a --environment=e --root-key=test"))
	require.ErrorContains(t, err, "variables")
}
