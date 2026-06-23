package stripe

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/tui"
)

// resetCmd returns a workspace to a clean Free state for billing tests.
//
// Test clocks cannot run backwards, so "rewinding" a billing test means
// tearing down and starting over: delete the Stripe test clock (which removes
// its customers and their subscriptions), then clear the workspace's billing
// columns and quota the way the subscription.deleted webhook would, plus the
// customer id, which no webhook ever clears. Afterwards a fresh clocked
// customer is one add-payment-method away.
var resetCmd = &cli.Command{
	Name:  "reset",
	Usage: "Reset a workspace's billing state: delete its Stripe customer/test clock and return the DB to Free",
	Flags: []cli.Flag{
		keyFlag(),
		cli.String("workspace", "Workspace ID to reset", cli.Required()),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.Bool("keep-stripe", "Leave the Stripe customer / test clock in place; only reset the database"),
	},
	Action: reset,
}

func reset(ctx context.Context, cmd *cli.Command) error {
	workspaceID := cmd.RequireString("workspace")
	out := tui.New(os.Stdout)

	database, err := db.New(db.Config{PrimaryDSN: cmd.RequireString("database-primary"), ReadOnlyDSN: ""})
	if err != nil {
		return fmt.Errorf("connect to MySQL: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	ws, err := db.Query.FindWorkspaceByID(ctx, database.RW(), workspaceID)
	if err != nil {
		return fmt.Errorf("find workspace %s: %w", workspaceID, err)
	}

	if !cmd.Bool("keep-stripe") && ws.StripeCustomerID.Valid && ws.StripeCustomerID.String != "" {
		sc, err := newClient(cmd)
		if err != nil {
			return err
		}
		customerID := ws.StripeCustomerID.String
		customer, err := sc.V1Customers.Retrieve(ctx, customerID, nil)
		switch {
		case err != nil && isResourceMissing(err):
			out.Println(out.Dim(fmt.Sprintf("Customer %s is already gone in Stripe.", customerID)))
		case err != nil:
			// Unexpected Stripe trouble; the DB reset below is still worth doing.
			out.Println(out.Yellow(fmt.Sprintf("Skipping Stripe cleanup, could not retrieve %s: %v", customerID, err)))
		case customer.Deleted:
			// Retrieving a deleted customer returns a stub, not a 404.
			out.Println(out.Dim(fmt.Sprintf("Customer %s was already deleted in Stripe.", customerID)))
		case customer.TestClock != nil:
			// Deleting the clock deletes every customer on it, subscriptions included.
			if _, err := sc.V1TestHelpersTestClocks.Delete(ctx, customer.TestClock.ID, nil); err != nil && !isResourceMissing(err) {
				return fmt.Errorf("delete test clock %s: %w", customer.TestClock.ID, err)
			}
			out.Printf("Deleted test clock %s (and customer %s with it)\n", customer.TestClock.ID, customerID)
		default:
			if _, err := sc.V1Customers.Delete(ctx, customerID, nil); err != nil && !isResourceMissing(err) {
				return fmt.Errorf("delete customer %s: %w", customerID, err)
			}
			out.Printf("Deleted customer %s\n", customerID)
		}
	}

	// Mirror the subscription.deleted webhook's reset, plus stripe_customer_id,
	// which the webhook leaves in place.
	if err := db.Query.ResetWorkspaceBilling(ctx, database.RW(), workspaceID); err != nil {
		return fmt.Errorf("reset workspace row: %w", err)
	}

	// Free-tier quota values, mirroring freeTierQuotas in the dashboard
	// (web/apps/dashboard/lib/quotas.ts). Every column is set, not just the core
	// quotas, so a previously paid workspace does not keep elevated rate-limit or
	// Deploy-resource allowances.
	if err := db.Query.UpdateQuota(ctx, database.RW(), db.UpdateQuotaParams{
		RequestsPerMonth:            150_000,
		LogsRetentionDays:           7,
		AuditLogsRetentionDays:      30,
		Team:                        false,
		RatelimitApiLimit:           sql.NullInt32{},
		RatelimitApiDuration:        sql.NullInt32{},
		AllocatedCpuMillicoresTotal: 10_000,
		AllocatedMemoryMibTotal:     20_480,
		AllocatedStorageMibTotal:    51_200,
		MaxCpuMillicoresPerInstance: 2_000,
		MaxMemoryMibPerInstance:     4_096,
		MaxStorageMibPerInstance:    10_240,
		MaxConcurrentBuilds:         1,
		WorkspaceID:                 workspaceID,
	}); err != nil {
		return fmt.Errorf("reset quota row: %w", err)
	}

	out.Println(out.Green(fmt.Sprintf("Workspace %s reset to Free.", workspaceID)))
	out.Println(out.Dim("Set STRIPE_DEV_TEST_CLOCK=true and add a payment method to start a fresh clocked customer."))
	return nil
}
