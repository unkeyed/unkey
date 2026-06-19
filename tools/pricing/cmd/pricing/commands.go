package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/unkeyed/unkey/tools/pricing"
	"github.com/unkeyed/unkey/tools/pricing/internal/export"
	"github.com/unkeyed/unkey/tools/pricing/internal/reconcile"
	"github.com/unkeyed/unkey/tools/pricing/internal/stripeenv"
)

func actionPlan(ctx context.Context, cmd *cli.Command) error {
	sc, err := clientFor(ctx, cmd)
	if err != nil {
		return err
	}
	res, _, err := reconcile.Run(ctx, sc, false, nil)
	if err != nil {
		return err
	}
	fmt.Print(res.Render(useColor(os.Stdout)))
	return nil
}

func actionApply(ctx context.Context, cmd *cli.Command) error {
	sc, err := clientFor(ctx, cmd)
	if err != nil {
		return err
	}
	color := useColor(os.Stdout)

	// Dry run first so the operator sees exactly what apply will do.
	plan, _, err := reconcile.Run(ctx, sc, false, nil)
	if err != nil {
		return err
	}
	fmt.Print(plan.Render(color))
	if !plan.HasChanges() {
		return nil
	}

	if sc.Env == pricing.EnvProduction && !cmd.Bool("yes") {
		if err := confirm(sc.Env); err != nil {
			return err
		}
	}

	// Stream each write as it lands so a mid-run failure shows what already
	// succeeded; the error names the object that did not.
	if plan.HasWrites() {
		fmt.Printf("\nApplying to %s:\n", sc.Env)
	}
	res, secrets, err := reconcile.Run(ctx, sc, true, func(c reconcile.Change) {
		fmt.Println(c.Line(color))
	})
	if err != nil {
		return err
	}
	fmt.Println("\n" + res.ApplySummary())

	for key, secret := range secrets {
		fmt.Printf("\nwebhook %q signing secret (shown once, store it now):\n%s\n", key, secret)
	}
	return nil
}

func actionVerify(ctx context.Context, cmd *cli.Command) error {
	sc, err := clientFor(ctx, cmd)
	if err != nil {
		return err
	}
	res, _, err := reconcile.Run(ctx, sc, false, nil)
	if err != nil {
		return err
	}
	fmt.Print(res.Render(useColor(os.Stdout)))
	if res.HasChanges() {
		return fmt.Errorf("stripe drifts from catalog: %d change(s)", res.CountChanges())
	}
	return nil
}

func actionExport(ctx context.Context, cmd *cli.Command) error {
	sc, err := clientFor(ctx, cmd)
	if err != nil {
		return err
	}
	block, err := export.Render(ctx, sc, nil)
	if err != nil {
		return err
	}
	fmt.Print(block)
	return nil
}

// clientFor resolves the --env flag and builds a Stripe client for it.
func clientFor(ctx context.Context, cmd *cli.Command) (*stripeenv.Client, error) {
	env, err := parseEnv(cmd.String("env"))
	if err != nil {
		return nil, err
	}
	return stripeenv.New(ctx, env)
}

func parseEnv(s string) (pricing.Environment, error) {
	switch pricing.Environment(s) {
	case pricing.EnvSandbox, pricing.EnvCanary, pricing.EnvProduction:
		return pricing.Environment(s), nil
	default:
		return "", fmt.Errorf("invalid --env %q (want sandbox | canary | production)", s)
	}
}

func confirm(env pricing.Environment) error {
	fmt.Printf("\nThis will modify the %s Stripe account. Type %q to continue: ", env, env)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != string(env) {
		return fmt.Errorf("aborted")
	}
	return nil
}

// useColor enables ANSI color only on a real terminal with NO_COLOR unset, so
// CI logs (verify) and piped output stay plain text.
func useColor(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
