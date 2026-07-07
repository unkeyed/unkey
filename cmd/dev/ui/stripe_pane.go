package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	devstripe "github.com/unkeyed/unkey/internal/devtools/stripe"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

type stripePane struct {
	rows          []devstripe.ClockRow
	cursor        int
	width         int
	height        int
	loading       bool
	busy          bool
	busyText      string
	invoices      []devstripe.InvoiceSummary
	showInv       bool
	statusMessage string
	spinner       app.Spinner
	stripeConfirm *stripeConfirm
}

func newStripePane() *stripePane {
	s := app.NewSpinner()
	return &stripePane{ //nolint:exhaustruct
		spinner: s,
		loading: true,
	}
}

func (p *stripePane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *stripePane) InitCmd() app.Cmd { return p.refreshCmd() }

func (p *stripePane) refreshCmd() app.Cmd {
	p.loading = true
	return withSpinnerTick(p.spinner, func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sc, err := devstripe.NewClient(os.Getenv("STRIPE_SECRET_KEY"))
		if err != nil {
			return stripeRowsMsg{rows: nil, err: err}
		}
		rows, err := devstripe.ListClockRows(ctx, sc)
		return stripeRowsMsg{rows: rows, err: err}
	})
}

type stripeRowsMsg struct {
	rows []devstripe.ClockRow
	err  error
}

type stripeDoneMsg struct {
	err      error
	message  string
	invoices map[string][]devstripe.InvoiceSummary
}

type invoiceMsg struct {
	customerID string
	invoices   []devstripe.InvoiceSummary
}

func (p *stripePane) Dispatch(msg app.Msg) (bool, app.Cmd) {
	switch msg := msg.(type) {
	case stripeRowsMsg:
		p.loading = false
		p.busy = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
			return true, nil
		}
		p.rows = msg.rows
		p.cursor = moveCursor(p.cursor, 0, len(p.rows))
		p.statusMessage = fmt.Sprintf("%d clocks", len(p.rows))
		return true, nil
	case stripeDoneMsg:
		p.busy = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
		} else {
			p.statusMessage = msg.message
			if len(msg.invoices) > 0 {
				p.showInv = true
				for customerID, invs := range msg.invoices {
					p.invoices = invs
					p.statusMessage = "Advanced · invoices for " + customerID
					break
				}
			}
		}
		return true, p.refreshCmd()
	case invoiceMsg:
		p.invoices = msg.invoices
		p.showInv = true
		p.statusMessage = "Invoices for " + msg.customerID
		return true, nil
	case app.SpinnerTickMsg:
		if p.busy || p.loading {
			var cmd app.Cmd
			var handled bool
			p.spinner, cmd, handled = handleSpinnerTick(p.spinner, msg)
			if handled {
				return true, cmd
			}
		}
	}
	return false, nil
}

func (p *stripePane) Update(msg app.Msg) app.Cmd {
	if p.stripeConfirm != nil {
		if key, ok := msg.(app.KeyMsg); ok {
			return p.updateStripeConfirm(key)
		}
		return nil
	}
	key, ok := msg.(app.KeyMsg)
	if !ok || p.busy {
		return nil
	}
	switch {
	case keyUp(key):
		p.cursor = moveCursor(p.cursor, -1, len(p.rows))
		return nil
	case keyDown(key):
		p.cursor = moveCursor(p.cursor, 1, len(p.rows))
		return nil
	}
	if cmd, ok := p.keys().handle(key); ok {
		return cmd
	}
	return nil
}

func (p *stripePane) keys() keyList {
	return keyList{
		{bind: app.Binding{Keys: []string{"a"}, Help: "advance"}, do: func() app.Cmd {
			p.openStripeConfirm(stripeActionAdvance)
			return nil
		}},
		{bind: app.Binding{Keys: []string{"d"}, Help: "delete"}, do: func() app.Cmd {
			p.openStripeConfirm(stripeActionDelete)
			return nil
		}},
		{bind: app.Binding{Keys: []string{"i"}, Help: "invoices"}, do: func() app.Cmd {
			row, ok := p.selectedRow()
			if !ok || row.CustomerID == "" {
				p.statusMessage = "no customer on this row"
				return nil
			}
			return p.loadInvoices(row.CustomerID)
		}},
		{bind: app.Binding{Keys: []string{"r"}, Help: "refresh"}, do: func() app.Cmd {
			p.showInv = false
			return p.refreshCmd()
		}},
	}
}

func (p *stripePane) Status() string { return p.statusMessage }

func (p *stripePane) Summary() string {
	if len(p.rows) == 0 {
		return ""
	}
	return strconv.Itoa(len(p.rows)) + " clocks"
}

func (p *stripePane) DetailView(width int) string {
	row, ok := p.selectedRow()
	if !ok {
		return renderDetail(width, "Test clocks", "none loaded", nil)
	}
	status := string(row.Status)
	sstyle := detailMidTone
	switch status {
	case "ready":
		sstyle = detailOK
	case "internal_failure":
		sstyle = detailBad
	}
	fields := []detailField{
		{Label: "status", Value: status, Style: sstyle},
		{Label: "frozen", Value: devstripe.FormatTime(row.FrozenTime), Style: detailPlain},
		{Label: "customer", Value: row.CustomerID, Style: detailPlain},
		{Label: "workspace", Value: orDash(row.WorkspaceID), Style: detailPlain},
	}
	if row.HasSubscription {
		fields = append(fields, detailField{Label: "period", Value: devstripe.FormatTime(row.PeriodEnd), Style: detailPlain})
	}
	return renderDetail(width, row.ClockID, "test clock", fields)
}

func (p *stripePane) KeyHints() []string {
	if p.stripeConfirm != nil {
		return p.confirmKeyHints()
	}
	return p.keys().hints()
}

func (p *stripePane) selectedRow() (devstripe.ClockRow, bool) {
	if p.cursor < 0 || p.cursor >= len(p.rows) {
		return devstripe.ClockRow{}, false //nolint:exhaustruct
	}
	return p.rows[p.cursor], true
}

func (p *stripePane) advanceRow(row devstripe.ClockRow, opts devstripe.AdvanceOptions) app.Cmd {
	p.busy = true
	p.busyText = "Advancing..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		sc, err := devstripe.NewClient(os.Getenv("STRIPE_SECRET_KEY"))
		if err != nil {
			return stripeErr(err)
		}
		err = devstripe.AdvanceClock(ctx, sc, row.ClockID, opts, nil)
		if err != nil {
			return stripeErr(err)
		}
		invoices, err := devstripe.ListInvoicesForClock(ctx, sc, row.ClockID)
		if err != nil {
			return stripeOK("Advanced "+row.ClockID, nil)
		}
		return stripeOK("Advanced "+row.ClockID, invoices)
	})
}

func (p *stripePane) deleteRow(row devstripe.ClockRow) app.Cmd {
	p.busy = true
	p.busyText = "Deleting..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sc, err := devstripe.NewClient(os.Getenv("STRIPE_SECRET_KEY"))
		if err != nil {
			return stripeErr(err)
		}
		_, err = devstripe.DeleteClock(ctx, sc, row.ClockID)
		if err != nil {
			return stripeErr(err)
		}
		return stripeOK("Deleted "+row.ClockID, nil)
	})
}

func (p *stripePane) loadInvoices(customerID string) app.Cmd {
	return func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sc, err := devstripe.NewClient(os.Getenv("STRIPE_SECRET_KEY"))
		if err != nil {
			return stripeErr(err)
		}
		invoices, err := devstripe.ListInvoices(ctx, sc, customerID)
		if err != nil {
			return stripeErr(err)
		}
		return invoiceMsg{customerID: customerID, invoices: invoices}
	}
}

func stripeErr(err error) stripeDoneMsg {
	return stripeDoneMsg{err: err, message: "", invoices: nil}
}

func stripeOK(message string, invoices map[string][]devstripe.InvoiceSummary) stripeDoneMsg {
	return stripeDoneMsg{err: nil, message: message, invoices: invoices}
}

func (p *stripePane) View() string {
	if p.stripeConfirm != nil {
		return p.confirmView()
	}
	if p.loading {
		return p.spinner.View() + " " + dimStyle.Render("Loading clocks...")
	}

	var b strings.Builder
	if p.busy {
		b.WriteString(p.spinner.View() + " " + p.busyText + "\n\n")
	}
	if len(p.rows) == 0 {
		b.WriteString(warnStyle.Render("No test clocks.") + "\n" +
			dimStyle.Render("Subscribe with STRIPE_DEV_TEST_CLOCK=true, then refresh."))
		return b.String()
	}
	cols := []tableColumn{
		{Title: "Clock", Width: 14},
		{Title: "Status", Width: 10},
		{Title: "Frozen", Width: 20},
		{Title: "Customer", Width: 18, Dim: true},
		{Title: "Workspace", Width: 14, Dim: true},
		{Title: "Period", Width: 12},
	}
	rows := make([][]string, 0, len(p.rows))
	for _, row := range p.rows {
		status := string(row.Status)
		period := "—"
		if row.HasSubscription {
			period = devstripe.FormatTime(row.PeriodEnd)
		}
		ws := row.WorkspaceID
		if ws == "" {
			ws = "—"
		}
		rows = append(rows, []string{
			row.ClockID,
			status,
			devstripe.FormatTime(row.FrozenTime),
			row.CustomerID,
			ws,
			period,
		})
	}
	b.WriteString(renderTable(p.width, cols, rows, p.cursor))
	if p.showInv && len(p.invoices) > 0 {
		b.WriteString("\n" + tableHeaderStyle.Render("Invoices") + "\n")
		for _, inv := range p.invoices {
			b.WriteString(normalRowStyle.Render(fmt.Sprintf("  %s %s %.2f %s\n",
				inv.ID, inv.Status, float64(inv.Total)/100, inv.Currency)))
		}
	}
	return b.String()
}
