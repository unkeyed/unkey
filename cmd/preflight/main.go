// Package preflight is the CLI entry point for the preflight E2E
// deploy-pipeline probe. All substantive logic lives in
// svc/preflight; this package only loads the TOML config and hands
// it over.
//
// See svc/preflight/README.md for the long-form overview,
// cmd/preflight/unkey.toml.example for a sample config, and
// docs/preflight/architecture.md for the system diagram.
package preflight

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/preflight"
)

// Cmd is the preflight sub-command exposed by the root unkey CLI.
// See main.go at the repo root where it is wired alongside `run` and
// `deploy`.
var Cmd = &cli.Command{
	Name:  "preflight",
	Usage: "Run the preflight E2E deploy-pipeline probe",
	Description: `Preflight exercises the full deploy pipeline against a target control
plane and asserts that every stage is healthy: webhook ingest, Depot build,
Krane provisioning, Vault env decrypt, Cilium policy, frontline routing,
sentinel request logs, and more.

Configuration is loaded from a TOML file. Secrets are NOT embedded in
the file directly; use ${VAR} interpolation and populate the
corresponding env vars via ExternalSecret / Kubernetes Secret at
runtime. See cmd/preflight/unkey.toml.example for the canonical shape.

The 'dev' target is implemented by the in-process harness at
svc/preflight/harness and is only reachable via ` + "`go test`:" + `

    go test -run TestDev ./svc/preflight/harness/...

Running 'preflight --target=dev' as a binary prints a hint and
exits non-zero. Keeping the binary staging/prod-only means we never
have to shim testing.T in production paths.

Intended to be invoked as a Kubernetes CronJob in each region; see
infra/eks-cluster/helm-chart/preflight/ for the deployment.`,
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"),
			cli.EnvVar("UNKEY_CONFIG"),
		),

		cli.String("config-data", "Inline TOML config content (takes precedence over --config)",
			cli.EnvVar("UNKEY_CONFIG_DATA"),
		),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	var (
		cfg preflight.Config
		err error
	)

	if data := cmd.String("config-data"); data != "" {
		cfg, err = config.LoadBytes[preflight.Config]([]byte(data))
	} else {
		cfg, err = config.Load[preflight.Config](cmd.String("config"))
	}
	if err != nil {
		return fmt.Errorf("preflight: load config: %w", err)
	}

	return preflight.Run(ctx, cfg)
}
