package stripe

import (
	"context"
	"fmt"
	"os"

	stripesdk "github.com/stripe/stripe-go/v86"
	devstripe "github.com/unkeyed/unkey/internal/devtools/stripe"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tui"
)

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
	rows, err := devstripe.ListClockRows(ctx, sc)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		out.Println("No test clocks.")
		out.Println(out.Dim("Set STRIPE_DEV_TEST_CLOCK=true in apps/dashboard/.env and add a payment method via the dashboard."))
		return nil
	}

	var lastClock string
	for _, row := range rows {
		if row.ClockID != lastClock {
			lastClock = row.ClockID
			title := out.Bold(row.ClockID)
			if row.ClockName != "" {
				title = out.Bold(row.ClockName) + "  " + out.Dim(row.ClockID)
			}
			out.Blank()
			out.Println(title)
			out.KV().Indent(2).
				Add("status", clockStatusLabel(out, row.Status)).
				Add("frozen at", devstripe.FormatTime(row.FrozenTime)).
				Print()

			table := out.Table("CUSTOMER", "WORKSPACE", "PERIOD ENDS").Indent(2)
			hasCustomers := false
			for _, r := range rows {
				if r.ClockID != row.ClockID || r.CustomerID == "" {
					continue
				}
				hasCustomers = true
				periodEnd := out.Dim("no subscription")
				if r.HasSubscription {
					periodEnd = devstripe.FormatTime(r.PeriodEnd)
				}
				table.Row(r.CustomerID, r.WorkspaceID, periodEnd)
			}
			if hasCustomers {
				out.Blank()
				table.Print()
			}
		}
	}
	return nil
}

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

	clockID, err := devstripe.ResolveClockID(ctx, sc, cmd.String("clock"), cmd.String("customer"))
	if err != nil {
		return err
	}
	clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, clockID, nil)
	if err != nil {
		return fmt.Errorf("retrieve clock %s: %w", clockID, err)
	}

	opts := devstripe.AdvanceOptions{ToRFC3339: cmd.String("to"), Hours: cmd.Float("hours")}
	target, err := devstripe.ResolveTargetTime(ctx, sc, clock, opts)
	if err != nil {
		return err
	}

	out := tui.New(os.Stdout)
	out.Printf("Advancing %s: %s -> %s ...\n", out.Bold(clockID), devstripe.FormatTime(clock.FrozenTime), devstripe.FormatTime(target))
	err = devstripe.AdvanceClock(ctx, sc, clockID, opts, func(p devstripe.AdvanceProgress) {
		if p.Done {
			out.Println(out.Green(fmt.Sprintf("Clock ready at %s.", devstripe.FormatTime(p.Frozen))))
			return
		}
		out.Println(out.Dim("  " + p.Status + "..."))
	})
	if err != nil {
		return err
	}

	for _, customerID := range customerIDsOnClock(rowsForClock(ctx, sc, clockID)) {
		out.Blank()
		out.Println(out.Bold("Invoices for " + customerID))
		if err := printInvoices(ctx, out, sc, customerID); err != nil {
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
	clockID, err := devstripe.ResolveClockID(ctx, sc, cmd.String("clock"), cmd.String("customer"))
	if err != nil {
		return err
	}
	customerIDs, err := devstripe.DeleteClock(ctx, sc, clockID)
	if err != nil {
		return err
	}
	out := tui.New(os.Stdout)
	deleted := "Deleted clock " + clockID
	for _, customerID := range customerIDs {
		deleted += ", customer " + customerID
	}
	out.Println(deleted)
	out.Println(out.Dim("If a workspace still references a deleted customer, run `unkey dev stripe reset --workspace <id>`."))
	return nil
}

func rowsForClock(ctx context.Context, sc *stripesdk.Client, clockID string) []devstripe.ClockRow {
	rows, err := devstripe.ListClockRows(ctx, sc)
	if err != nil {
		return nil
	}
	var filtered []devstripe.ClockRow
	for _, row := range rows {
		if row.ClockID == clockID {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func customerIDsOnClock(rows []devstripe.ClockRow) []string {
	var ids []string
	for _, row := range rows {
		if row.CustomerID != "" {
			ids = append(ids, row.CustomerID)
		}
	}
	return ids
}
