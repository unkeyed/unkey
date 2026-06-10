package stripe

import (
	"context"
	"fmt"
	"os"

	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tui"
)

// invoicesCmd lists a customer's recent invoices with their hosted page and
// PDF links. Drafts carry neither; finalized invoices carry both.
var invoicesCmd = &cli.Command{
	Name:  "invoices",
	Usage: "List a customer's recent invoices with hosted/PDF links",
	Flags: []cli.Flag{
		keyFlag(),
		cli.String("customer", "Customer id (cus_...)", cli.Required()),
	},
	Action: invoices,
}

func invoices(ctx context.Context, cmd *cli.Command) error {
	sc, err := newClient(cmd)
	if err != nil {
		return err
	}
	return printInvoices(ctx, tui.New(os.Stdout), sc, cmd.RequireString("customer"))
}

func printInvoices(ctx context.Context, out *tui.Renderer, sc *stripesdk.Client, customerID string) error {
	list := sc.V1Invoices.List(ctx, &stripesdk.InvoiceListParams{
		ListParams: stripesdk.ListParams{Limit: stripesdk.Int64(5)},
		Customer:   stripesdk.String(customerID),
	})
	count := 0
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return fmt.Errorf("list invoices: %w", err)
		}
		count++
		out.Blank()
		out.Printf("%s  %s  %.2f %s\n",
			out.Bold(invoice.ID), invoiceStatusLabel(out, invoice.Status), float64(invoice.Total)/100, invoice.Currency)
		out.KV().Indent(2).
			Add("period", formatTime(invoice.PeriodStart)+" -> "+formatTime(invoice.PeriodEnd)).
			AddIf("hosted", out.Cyan(invoice.HostedInvoiceURL)).
			AddIf("pdf", out.Cyan(invoice.InvoicePDF)).
			Print()
		if invoice.InvoicePDF == "" && invoice.Status == stripesdk.InvoiceStatusDraft {
			out.Println(out.Dim("  draft: advance the clock ~2h further to finalize"))
		}
	}
	if count == 0 {
		out.Println("No invoices yet.")
	}
	return nil
}

// invoiceStatusLabel colors an invoice status by outcome: paid is done, draft
// and open are still moving, void and uncollectible are dead ends.
func invoiceStatusLabel(out *tui.Renderer, status stripesdk.InvoiceStatus) string {
	switch status {
	case stripesdk.InvoiceStatusPaid:
		return out.Green(string(status))
	case stripesdk.InvoiceStatusDraft, stripesdk.InvoiceStatusOpen:
		return out.Yellow(string(status))
	case stripesdk.InvoiceStatusVoid, stripesdk.InvoiceStatusUncollectible:
		return out.Red(string(status))
	default:
		return out.Red(string(status))
	}
}
