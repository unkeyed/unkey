package workflows

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	server "github.com/restatedev/sdk-go/server"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/workflows"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:        "workflows",
	Description: "Run a workflow worker",
	Flags: []cli.Flag{

		&cli.StringFlag{
			Name:     "database-primary",
			Usage:    "DSN for the primary database",
			Sources:  cli.EnvVars("UNKEY_DATABASE_PRIMARY"),
			Required: true,
		},
		&cli.StringFlag{
			Name:    "heartbeat-url",
			Usage:   "Healthcheck URL to send heartbeats",
			Sources: cli.EnvVars("UNKEY_HEARTBEAT_URL"),
		},
		&cli.IntFlag{
			Name:    "http-port",
			Usage:   "Port to bind the HTTP server to",
			Sources: cli.EnvVars("UNKEY_HTTP_PORT"),
			Value:   9080,
		},
		&cli.StringFlag{
			Name:    "public-key",
			Usage:   "Identity key to validate incoming requests",
			Sources: cli.EnvVars("UNKEY_PUBLIC_KEY"),
		},
		&cli.StringFlag{
			Name:    "refill-heartbeat-url",
			Usage:   "Healthcheck URL to send heartbeats",
			Sources: cli.EnvVars("UNKEY_REFILL_HEARTBEAT_URL"),
		},
		&cli.StringFlag{
			Name:    "count-keys-heartbeat-url",
			Usage:   "Healthcheck URL to send heartbeats",
			Sources: cli.EnvVars("UNKEY_COUNT_KEYS_HEARTBEAT_URL"),
		},
	},
	Action: run,
}

// nolint:gocognit
func run(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.String("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return err
	}

	audit := auditlogs.New(auditlogs.Config{
		DB:     database,
		Logger: logger,
	})

	srv := server.NewRestate()

	srv.Bind(restate.Reflect(&workflows.RefillWorkflow{
		DB:           database,
		Audit:        audit,
		HeartbeatURL: cmd.String("refill-heartbeat-url"),
	}))
	srv.Bind(restate.Reflect(&workflows.CountKeysWorkflow{
		DB:           database,
		HeartbeatURL: cmd.String("count-keys-heartbeat-url"),
	}))

	identityV1 := cmd.String("public-key")
	if identityV1 != "" {

		srv.WithIdentityV1(identityV1)
	}
	err = srv.Start(ctx, fmt.Sprintf(":%d", cmd.Int("http-port")))

	if err != nil {
		return err
	}

	return nil
}
