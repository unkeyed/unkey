package stripe

import (
	"context"
	"fmt"
	"os"

	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tui"
)

// grantsCmd lists a customer's billing credit grants and their available
// metered-usage balance: the fastest way to answer "where are my credits?".
// A missing grant usually means the invoice.payment_succeeded webhook never
// reached the dashboard (is `stripe listen` running?); a present grant with
// an untouched balance just has not been applied yet, which happens at
// invoice finalization.
var grantsCmd = &cli.Command{
	Name:  "grants",
	Usage: "List a customer's credit grants and available balance",
	Flags: []cli.Flag{
		keyFlag(),
		cli.String("customer", "Customer id (cus_...)", cli.Required()),
	},
	Action: grants,
}

func grants(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}
	customerID := cmd.RequireString("customer")
	out := tui.New(os.Stdout)

	count := 0
	list := sc.V1BillingCreditGrants.List(ctx, &stripesdk.BillingCreditGrantListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(20)},
		Customer:   stripesdk.String(customerID),
	})
	for grant, err := range list.All(ctx) {
		if err != nil {
			return fmt.Errorf("list credit grants: %w", err)
		}
		count++
		amount := ""
		if grant.Amount != nil && grant.Amount.Monetary != nil {
			amount = fmt.Sprintf("%.2f %s", float64(grant.Amount.Monetary.Value)/100, grant.Amount.Monetary.Currency)
		}
		out.Blank()
		out.Printf("%s  %s  %s\n", out.Bold(grant.ID), grant.Name, amount)
		kv := out.KV().Indent(2)
		if grant.ExpiresAt > 0 {
			kv.Add("expires", formatTime(grant.ExpiresAt))
		}
		if grant.VoidedAt > 0 {
			kv.Add("voided", out.Red(formatTime(grant.VoidedAt)))
		}
		kv.AddIf("invoice", grant.Metadata["stripe_invoice_id"]).Print()
	}
	if count == 0 {
		out.Println("No credit grants.")
		out.Println(out.Dim("Grants are created by the invoice.payment_succeeded webhook; make sure `stripe listen` forwards events to the dashboard."))
		return nil
	}

	// Available balance against metered prices, the scope our grants use.
	summary, err := sc.V1BillingCreditBalanceSummary.Retrieve(ctx, &stripesdk.BillingCreditBalanceSummaryRetrieveParams{
		Customer: stripesdk.String(customerID),
		Filter: &stripesdk.BillingCreditBalanceSummaryRetrieveFilterParams{
			Type: stripesdk.String("applicability_scope"),
			ApplicabilityScope: &stripesdk.BillingCreditBalanceSummaryRetrieveFilterApplicabilityScopeParams{
				PriceType: stripesdk.String("metered"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("retrieve credit balance summary: %w", err)
	}
	out.Blank()
	for _, balance := range summary.Balances {
		if balance.AvailableBalance != nil && balance.AvailableBalance.Monetary != nil {
			out.Printf("Available balance: %s\n", out.Green(fmt.Sprintf("%.2f %s",
				float64(balance.AvailableBalance.Monetary.Value)/100, balance.AvailableBalance.Monetary.Currency)))
		}
	}
	return nil
}
