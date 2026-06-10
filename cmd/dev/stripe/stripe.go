// Package stripe groups Stripe development tools: test-clock time travel for
// billing tests, invoice inspection, and whatever billing tooling comes next.
//
// Everything here is dev/test only and refuses to run against live keys.
package stripe

import (
	"errors"
	"fmt"
	"strings"
	"time"

	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/cli"
)

// isResourceMissing reports whether err is Stripe's 404 for an object that no
// longer exists. The dev tools treat that as "already done" rather than a
// failure: deletes and resets must be idempotent to be useful for cleanup.
func isResourceMissing(err error) bool {
	var stripeErr *stripesdk.Error
	return errors.As(err, &stripeErr) && stripeErr.Code == stripesdk.ErrorCodeResourceMissing
}

// Cmd is the stripe dev-tools namespace.
var Cmd = &cli.Command{
	Name:  "stripe",
	Usage: "Stripe development tools",
	Commands: []*cli.Command{
		clockCmd,
		invoicesCmd,
		grantsCmd,
		resetCmd,
	},
}

func keyFlag() cli.Flag {
	return cli.String(
		"stripe-secret-key",
		"Stripe test-mode secret key",
		cli.EnvVar("STRIPE_SECRET_KEY"),
		cli.Required(),
	)
}

// newClient builds a Stripe client, refusing live keys: these tools mutate
// billing state and exist for local testing only.
func newClient(cmd *cli.Command) (*stripesdk.Client, error) {
	key := cmd.RequireString("stripe-secret-key")
	if !strings.HasPrefix(key, "sk_test_") && !strings.HasPrefix(key, "rk_test_") {
		return nil, fmt.Errorf("refusing to run: the Stripe key is not a test-mode key")
	}

	return stripesdk.NewClient(key, stripesdk.WithBackends(stripesdk.NewBackendsWithConfig(&stripesdk.BackendConfig{
		//nolint:exhaustruct // defaults are fine for everything but the logger
		// Every error is returned and rendered by the command itself; the
		// SDK's default stderr printer would double-report them, raw and ugly.
		LeveledLogger: &stripesdk.LeveledLogger{Level: stripesdk.LevelNull},
	}))), nil
}

func formatTime(unixSeconds int64) string {
	return time.Unix(unixSeconds, 0).UTC().Format(time.RFC3339)
}
