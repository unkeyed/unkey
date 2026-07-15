package stripe

import (
	"context"
	"fmt"
	"os"
	"time"

	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tui"
)

// clockCmd is time travel for Stripe billing tests.
//
// It works with customers created under a Stripe test clock (set
// STRIPE_DEV_TEST_CLOCK=true in apps/dashboard/.env before adding a payment
// method via the dashboard UI; every checkout then pre-creates a clocked
// customer). Advancing a clock past the subscription's period end makes Stripe
// generate AND finalize the renewal invoice for real, so it carries a hosted
// page and a PDF, with credit grants applied.
//
// Subscription invoices auto-finalize about one hour after creation, so the
// default advance target is period end + 2 hours: one jump from "mid-month" to
// "finalized invoice with PDF".
var clockCmd = &cli.Command{
	Name:  "clock",
	Usage: "Time travel test clocks for billing tests",
	Commands: []*cli.Command{
		{
			Name:   "status",
			Usage:  "List test clocks, their customers, and period ends",
			Flags:  []cli.Flag{keyFlag()},
			Action: clockStatus,
		},
		{
			Name:  "advance",
			Usage: "Advance a clock (default: subscription period end + 2h, which finalizes the invoice)",
			Flags: []cli.Flag{
				keyFlag(),
				cli.String("customer", "Customer id (cus_...) whose clock to advance"),
				cli.String("clock", "Test clock id (tc_...) to advance"),
				cli.String("to", "Absolute target time, RFC3339 (e.g. 2026-07-01T03:00:00Z)"),
				cli.Float("hours", "Advance by this many hours from the clock's frozen time"),
			},
			RequireOneOf: [][]string{{"customer", "clock"}},
			Action:       clockAdvance,
		},
		{
			// Clocks cannot run backwards; deleting and starting over is the
			// only rewind. See `dev stripe reset` for the full workspace reset.
			Name:  "delete",
			Usage: "Delete a test clock (removes its customers and their subscriptions)",
			Flags: []cli.Flag{
				keyFlag(),
				cli.String("customer", "Customer id (cus_...) whose clock to delete"),
				cli.String("clock", "Test clock id (tc_...) to delete"),
			},
			RequireOneOf: [][]string{{"customer", "clock"}},
			Action:       clockDelete,
		},
	},
}

func clockStatus(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}

	out := tui.New(os.Stdout)
	found := false
	clocks := sc.V1TestHelpersTestClocks.List(ctx, &stripesdk.TestHelpersTestClockListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(20)},
	})
	for clock, err := range clocks.All(ctx) {
		if err != nil {
			return fmt.Errorf("list test clocks: %w", err)
		}
		found = true

		title := out.Bold(clock.ID)
		if clock.Name != "" {
			title = out.Bold(clock.Name) + "  " + out.Dim(clock.ID)
		}
		out.Blank()
		out.Println(title)
		out.KV().Indent(2).
			Add("status", clockStatusLabel(out, clock.Status)).
			Add("frozen at", formatTime(clock.FrozenTime)).
			Print()

		customers, err := listClockCustomers(ctx, sc, clock.ID)
		if err != nil {
			return err
		}
		if len(customers) == 0 {
			continue
		}
		out.Blank()
		table := out.Table("CUSTOMER", "WORKSPACE", "PERIOD ENDS").Indent(2)
		for _, customer := range customers {
			end, hasSub, err := latestPeriodEnd(ctx, sc, customer.ID)
			if err != nil {
				return err
			}
			periodEnd := out.Dim("no subscription")
			if hasSub {
				periodEnd = formatTime(end)
			}
			table.Row(customer.ID, customer.Metadata["workspace_id"], periodEnd)
		}
		table.Print()
	}
	if !found {
		out.Println("No test clocks.")
		out.Println(out.Dim("Set STRIPE_DEV_TEST_CLOCK=true in apps/dashboard/.env and add a payment method via the dashboard."))
	}
	return nil
}

// clockStatusLabel colors a clock status by how much attention it needs:
// ready is the steady state, advancing resolves on its own, anything else is
// a problem.
func clockStatusLabel(out *tui.Renderer, status stripesdk.TestHelpersTestClockStatus) string {
	switch status {
	case stripesdk.TestHelpersTestClockStatusReady:
		return out.Green(string(status))
	case stripesdk.TestHelpersTestClockStatusAdvancing:
		return out.Yellow(string(status))
	case stripesdk.TestHelpersTestClockStatusInternalFailure:
		return out.Red(string(status))
	default:
		return out.Red(string(status))
	}
}

func clockAdvance(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}

	clockID, err := resolveClockID(ctx, sc, cmd)
	if err != nil {
		return err
	}
	clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, clockID, nil)
	if err != nil {
		return fmt.Errorf("retrieve clock %s: %w", clockID, err)
	}

	target, err := targetTime(ctx, sc, cmd, clock)
	if err != nil {
		return err
	}
	if target <= clock.FrozenTime {
		return fmt.Errorf("target %s is not after the clock's %s", formatTime(target), formatTime(clock.FrozenTime))
	}

	out := tui.New(os.Stdout)
	out.Printf("Advancing %s: %s -> %s ...\n", out.Bold(clockID), formatTime(clock.FrozenTime), formatTime(target))
	_, err = sc.V1TestHelpersTestClocks.Advance(ctx, clockID, &stripesdk.TestHelpersTestClockAdvanceParams{
		FrozenTime: stripesdk.Int64(target),
	})
	if err != nil {
		return fmt.Errorf("advance clock: %w", err)
	}

	// Advancing is asynchronous; poll until the clock settles so the invoices
	// printed below actually exist.
	for {
		time.Sleep(2 * time.Second)
		current, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, clockID, nil)
		if err != nil {
			return fmt.Errorf("poll clock: %w", err)
		}
		if current.Status == stripesdk.TestHelpersTestClockStatusReady {
			out.Println(out.Green(fmt.Sprintf("Clock ready at %s.", formatTime(current.FrozenTime))))
			break
		}
		if current.Status == stripesdk.TestHelpersTestClockStatusInternalFailure {
			return fmt.Errorf("stripe reported an internal failure advancing the clock")
		}
		out.Println(out.Dim("  " + string(current.Status) + "..."))
	}

	customers, err := listClockCustomers(ctx, sc, clockID)
	if err != nil {
		return err
	}
	for _, customer := range customers {
		out.Blank()
		out.Println(out.Bold("Invoices for " + customer.ID))
		if err := printInvoices(ctx, out, sc, customer.ID); err != nil {
			return err
		}
	}
	return nil
}

func clockDelete(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}
	clockID, err := resolveClockID(ctx, sc, cmd)
	if err != nil {
		return err
	}
	customers, err := listClockCustomers(ctx, sc, clockID)
	if err != nil {
		return err
	}
	if _, err := sc.V1TestHelpersTestClocks.Delete(ctx, clockID, nil); err != nil {
		return fmt.Errorf("delete clock %s: %w", clockID, err)
	}
	out := tui.New(os.Stdout)
	deleted := "Deleted clock " + clockID
	for _, customer := range customers {
		deleted += ", customer " + customer.ID
	}
	out.Println(deleted)
	out.Println(out.Dim("If a workspace still references a deleted customer, run `unkey dev stripe reset --workspace <id>`."))
	return nil
}

// resolveClockID returns the clock to act on: --clock directly, or the clock
// the --customer lives on. The RequireOneOf constraint on the commands
// guarantees exactly one of the two flags is set.
func resolveClockID(ctx context.Context, sc *stripesdk.Client, cmd *cli.Command) (string, error) {
	if clock := cmd.String("clock"); clock != "" {
		return clock, nil
	}
	customerID := cmd.String("customer")
	customer, err := sc.V1Customers.Retrieve(ctx, customerID, nil)
	if err != nil {
		return "", fmt.Errorf("retrieve customer %s: %w", customerID, err)
	}
	if customer.TestClock == nil {
		return "", fmt.Errorf("customer %s is not on a test clock", customerID)
	}
	return customer.TestClock.ID, nil
}

// targetTime resolves the advance target: --to, --hours, or the latest
// subscription period end on the clock plus two hours (so the renewal invoice
// exists and has auto-finalized).
func targetTime(ctx context.Context, sc *stripesdk.Client, cmd *cli.Command, clock *stripesdk.TestHelpersTestClock) (int64, error) {
	if to := cmd.String("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return 0, fmt.Errorf("parse --to: %w", err)
		}
		return t.Unix(), nil
	}
	if hours := cmd.Float("hours"); hours > 0 {
		return clock.FrozenTime + int64(hours*3600), nil
	}

	customers, err := listClockCustomers(ctx, sc, clock.ID)
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, customer := range customers {
		end, ok, err := latestPeriodEnd(ctx, sc, customer.ID)
		if err != nil {
			return 0, err
		}
		if ok && end > latest {
			latest = end
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("no subscriptions on this clock; pass --to or --hours instead")
	}
	return latest + 2*3600, nil
}

func listClockCustomers(ctx context.Context, sc *stripesdk.Client, clockID string) ([]*stripesdk.Customer, error) {
	list := sc.V1Customers.List(ctx, &stripesdk.CustomerListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(5)},
		TestClock:  stripesdk.String(clockID),
	})
	var customers []*stripesdk.Customer
	for customer, err := range list.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list customers on clock %s: %w", clockID, err)
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

// latestPeriodEnd returns the latest current_period_end across the customer's
// subscription items (the period lives on the items), and whether any
// subscription exists at all.
func latestPeriodEnd(ctx context.Context, sc *stripesdk.Client, customerID string) (int64, bool, error) {
	list := sc.V1Subscriptions.List(ctx, &stripesdk.SubscriptionListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(10)},
		Customer:   stripesdk.String(customerID),
	})
	var latest int64
	found := false
	for sub, err := range list.All(ctx) {
		if err != nil {
			return 0, false, fmt.Errorf("list subscriptions for %s: %w", customerID, err)
		}
		for _, item := range sub.Items.Data {
			found = true
			if item.CurrentPeriodEnd > latest {
				latest = item.CurrentPeriodEnd
			}
		}
	}
	return latest, found, nil
}
