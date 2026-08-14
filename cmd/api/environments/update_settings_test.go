package environments

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateSettings(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsUpdateSettingsRequestBody
	}{{"omits partial fields", `environments update-settings --project=p --app=a --environment=e`, openapi.V2EnvironmentsUpdateSettingsRequestBody{Project: "p", App: "a", Environment: "e"}}, {"scalar options", `environments update-settings --project=p --app=a --environment=e --auto-deploy=false --port=8080 --v-cpus=0.5 --memory-mib=512 --shutdown-signal=SIGTERM --upstream-protocol=h2c`, openapi.V2EnvironmentsUpdateSettingsRequestBody{Project: "p", App: "a", Environment: "e", AutoDeploy: ptr.P(false), Port: ptr.P(8080), VCpus: ptr.P(0.5), MemoryMib: ptr.P(512), ShutdownSignal: ptr.P(openapi.SIGTERM), UpstreamProtocol: ptr.P(openapi.H2c)}}, {"clear slices", `environments update-settings --project=p --app=a --environment=e --watch-paths= --command=`, openapi.V2EnvironmentsUpdateSettingsRequestBody{Project: "p", App: "a", Environment: "e", WatchPaths: ptr.P([]string{}), Command: ptr.P([]string{})}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.CaptureRequest[openapi.V2EnvironmentsUpdateSettingsRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateSettingsRejectsUnrepresentableJSON(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want string
	}{
		{name: "null healthcheck", flag: "--healthcheck=null", want: "cannot be null"},
		{name: "null regions", flag: "--regions=null", want: "non-empty JSON array"},
		{name: "empty regions", flag: "--regions=[]", want: "at least one region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := "unkey environments update-settings --project=p --app=a --environment=e --root-key=test " + tt.flag
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			err := root.Run(context.Background(), strings.Fields(args))
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestUpdateSettingsRejectsInvalidEnums(t *testing.T) {
	tests := []string{"--shutdown-signal=SIGHUP", "--upstream-protocol=http2"}
	for _, flag := range tests {
		t.Run(flag, func(t *testing.T) {
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			err := root.Run(context.Background(), strings.Fields("unkey environments update-settings --project=p --app=a --environment=e --root-key=test "+flag))
			require.ErrorContains(t, err, "invalid enum value")
		})
	}
}
