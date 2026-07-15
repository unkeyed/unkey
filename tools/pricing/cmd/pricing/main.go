// Command pricing reconciles the Unkey Stripe billing catalog (declared in the
// pricing package) against a Stripe account.
//
//	pricing plan   --env sandbox     # show the diff, write nothing
//	pricing apply  --env sandbox     # make Stripe match the catalog
//	pricing verify --env sandbox     # exit non-zero if Stripe drifts from catalog
//	pricing export --env sandbox     # print the dashboard env block from live Stripe
//
// The Stripe key comes from STRIPE_SECRET_KEY, or AWS Secrets Manager for
// canary/production (see internal/stripeenv). Applying to production requires
// typing the environment name to confirm. Each command's logic lives in
// commands.go; this file is only the CLI wiring.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/urfave/cli/v3"

	"github.com/unkeyed/unkey/tools/pricing"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := rootCmd().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func rootCmd() *cli.Command {
	return &cli.Command{
		Name:  "pricing",
		Usage: "reconcile the Unkey Stripe billing catalog",
		Commands: []*cli.Command{
			{
				Name:   "plan",
				Usage:  "show the diff between the catalog and Stripe (no writes)",
				Flags:  []cli.Flag{envFlag()},
				Action: actionPlan,
			},
			{
				Name:   "apply",
				Usage:  "make Stripe match the catalog",
				Flags:  []cli.Flag{envFlag(), yesFlag()},
				Action: actionApply,
			},
			{
				Name:   "verify",
				Usage:  "exit non-zero if Stripe drifts from the catalog (CI gate)",
				Flags:  []cli.Flag{envFlag()},
				Action: actionVerify,
			},
			{
				Name:   "export",
				Usage:  "print the dashboard env block from live Stripe",
				Flags:  []cli.Flag{envFlag()},
				Action: actionExport,
			},
		},
	}
}

func envFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "env",
		Value: string(pricing.EnvSandbox),
		Usage: "target environment: sandbox | canary | production",
	}
}

func yesFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  "yes",
		Usage: "skip the interactive confirmation prompt",
	}
}
