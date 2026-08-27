package environments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func updateSettingsCmd() *cli.Command {
	shutdownSignals := []string{string(components.EnvironmentShutdownSignalSigterm), string(components.EnvironmentShutdownSignalSigint), string(components.EnvironmentShutdownSignalSigquit), string(components.EnvironmentShutdownSignalSigkill)}
	upstreamProtocols := []string{string(components.EnvironmentUpstreamProtocolHttp1), string(components.EnvironmentUpstreamProtocolH2c)}
	return &cli.Command{Name: "update-settings", Usage: "Update environment build and runtime settings", Description: "Update selected build, runtime, health check, and region settings for an environment. Request-building flags that you omit leave their stored values unchanged.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/update-settings" + util.Disclaimer, Examples: []string{`unkey api environments update-settings --project=payments --app=payments-api --environment=production --healthcheck='{"method":"GET","path":"/health"}' --regions='[{"name":"us-east-1","replicas":{"min":1,"max":2}}]'`}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("dockerfile", "Dockerfile path.", cli.MutuallyExclusive("body")), cli.String("root-directory", "Build root directory.", cli.MutuallyExclusive("body")), cli.String("build-command", "Build command.", cli.MutuallyExclusive("body")), cli.StringSlice("watch-paths", "Paths that trigger deployment.", cli.MutuallyExclusive("body")), cli.Bool("auto-deploy", "Whether pushes auto-deploy.", cli.MutuallyExclusive("body")), cli.Int64("port", "Container port.", cli.MutuallyExclusive("body")), cli.Float("v-cpus", "CPU allocation in vCPUs.", cli.MutuallyExclusive("body")), cli.Int64("memory-mib", "Memory allocation in MiB.", cli.MutuallyExclusive("body")), cli.Int64("storage-mib", "Storage allocation in MiB.", cli.MutuallyExclusive("body")), cli.StringSlice("command", "Container command.", cli.MutuallyExclusive("body")), cli.String("healthcheck", "Healthcheck configuration as JSON.", cli.MutuallyExclusive("body")), cli.Enum("shutdown-signal", "Container shutdown signal.", shutdownSignals, cli.MutuallyExclusive("body")), cli.Enum("upstream-protocol", "Protocol used to reach the container.", upstreamProtocols, cli.MutuallyExclusive("body")), cli.String("openapi-spec-path", "OpenAPI specification path.", cli.MutuallyExclusive("body")), cli.String("regions", "Region configuration as JSON.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Environments.UpdateSettings, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2EnvironmentsUpdateSettingsResponseBody)
		}
		send := func(req components.V2EnvironmentsUpdateSettingsRequestBody) error {
			res, err := client.Environments.UpdateSettings(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2EnvironmentsUpdateSettingsResponseBody)
		}
		req := components.V2EnvironmentsUpdateSettingsRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Dockerfile: nil, RootDirectory: nil, BuildCommand: nil, WatchPaths: nil, AutoDeploy: nil, Port: nil, VCpus: nil, MemoryMib: nil, StorageMib: nil, Command: nil, Healthcheck: nil, ShutdownSignal: nil, UpstreamProtocol: nil, OpenapiSpecPath: nil, Regions: nil}
		if v := cmd.String("dockerfile"); v != "" {
			req.Dockerfile = optionalnullable.From(&v)
		}
		if v := cmd.String("root-directory"); v != "" {
			req.RootDirectory = &v
		}
		if v := cmd.String("build-command"); v != "" {
			req.BuildCommand = optionalnullable.From(&v)
		}
		if cmd.FlagIsSet("watch-paths") {
			req.WatchPaths = cmd.StringSlice("watch-paths")
		}
		if cmd.FlagIsSet("auto-deploy") {
			req.AutoDeploy = ptr.P(cmd.Bool("auto-deploy"))
		}
		if cmd.FlagIsSet("port") {
			req.Port = ptr.P(cmd.Int64("port"))
		}
		if cmd.FlagIsSet("v-cpus") {
			req.VCpus = ptr.P(cmd.Float("v-cpus"))
		}
		if cmd.FlagIsSet("memory-mib") {
			req.MemoryMib = ptr.P(cmd.Int64("memory-mib"))
		}
		if cmd.FlagIsSet("storage-mib") {
			req.StorageMib = ptr.P(cmd.Int64("storage-mib"))
		}
		if cmd.FlagIsSet("command") {
			req.Command = cmd.StringSlice("command")
		}
		if raw := cmd.String("healthcheck"); raw != "" {
			if strings.TrimSpace(raw) == "null" {
				return fmt.Errorf("--healthcheck cannot be null with the current SDK")
			}
			var v components.EnvironmentHealthcheck
			if err := json.Unmarshal([]byte(raw), &v); err != nil {
				return fmt.Errorf("invalid JSON for --healthcheck: %w", err)
			}
			req.Healthcheck = optionalnullable.From(&v)
		}
		if v := cmd.Enum("shutdown-signal"); v != "" {
			x := components.EnvironmentShutdownSignal(v)
			req.ShutdownSignal = &x
		}
		if v := cmd.Enum("upstream-protocol"); v != "" {
			x := components.EnvironmentUpstreamProtocol(v)
			req.UpstreamProtocol = &x
		}
		if v := cmd.String("openapi-spec-path"); v != "" {
			req.OpenapiSpecPath = optionalnullable.From(&v)
		}
		if raw := cmd.String("regions"); raw != "" {
			if strings.TrimSpace(raw) == "null" {
				return fmt.Errorf("--regions must be a non-empty JSON array")
			}
			if err := json.Unmarshal([]byte(raw), &req.Regions); err != nil {
				return fmt.Errorf("invalid JSON for --regions: %w", err)
			}
			if len(req.Regions) == 0 {
				return fmt.Errorf("--regions must contain at least one region")
			}
		}
		return send(req)
	}}
}
