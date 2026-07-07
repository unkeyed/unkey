package ui

import (
	"strconv"

	devstripe "github.com/unkeyed/unkey/internal/devtools/stripe"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

type stripeActionKind int

const (
	stripeActionAdvance stripeActionKind = iota
	stripeActionDelete
)

type stripeConfirm struct {
	action stripeActionKind
	row    devstripe.ClockRow
	hours  int
	cursor int
}

var stripeHourOptions = []int{0, 1, 2, 4, 8, 24}

func (p *stripePane) openStripeConfirm(action stripeActionKind) {
	row, ok := p.selectedRow()
	if !ok {
		p.statusMessage = "select a clock row"
		return
	}
	c := &stripeConfirm{
		action: action,
		row:    row,
		hours:  2,
		cursor: 0,
	}
	if action == stripeActionAdvance {
		c.cursor = 3
	}
	p.stripeConfirm = c
}

func (p *stripePane) stripeConfirmLines() []confirmLine {
	if p.stripeConfirm == nil {
		return nil
	}
	c := p.stripeConfirm
	lines := []confirmLine{
		{Label: "clock", Value: truncate(c.row.ClockID, 20), Scope: true},
		{Label: "customer", Value: truncate(c.row.CustomerID, 20)},
		{Label: "frozen", Value: devstripe.FormatTime(c.row.FrozenTime)},
	}
	switch c.action {
	case stripeActionAdvance:
		lines = append(lines, confirmLine{
			Label:      "advance",
			Value:      formatStripeHours(c.hours),
			Adjustable: true,
		})
	case stripeActionDelete:
		lines = append(lines, confirmLine{Label: "warning", Value: "Deletes clock and customers"})
	}
	return lines
}

func formatStripeHours(hours int) string {
	if hours == 0 {
		return "billing period + 2h"
	}
	return strconv.Itoa(hours) + "h"
}

func (p *stripePane) adjustStripeConfirm(delta int) {
	if p.stripeConfirm == nil || p.stripeConfirm.action != stripeActionAdvance {
		return
	}
	idx := 0
	for i, h := range stripeHourOptions {
		if h == p.stripeConfirm.hours {
			idx = i
			break
		}
	}
	idx = clampInt(idx+delta, 0, len(stripeHourOptions)-1)
	p.stripeConfirm.hours = stripeHourOptions[idx]
}

func (p *stripePane) updateStripeConfirm(key app.KeyMsg) app.Cmd {
	if p.stripeConfirm == nil {
		return nil
	}
	lines := p.stripeConfirmLines()
	nav := navigateConfirm(key, p.stripeConfirm.cursor, lines)
	if nav.Esc {
		p.stripeConfirm = nil
		return nil
	}
	p.stripeConfirm.cursor = nav.Cursor
	if nav.Adjust != 0 {
		p.adjustStripeConfirm(nav.Adjust)
		return nil
	}
	changeScope, run := handleConfirmEnter(nav, lines)
	if changeScope {
		p.stripeConfirm = nil
		return nil
	}
	if !run {
		return nil
	}
	c := p.stripeConfirm
	p.stripeConfirm = nil
	switch c.action {
	case stripeActionAdvance:
		return p.advanceRow(c.row, devstripe.AdvanceOptions{ToRFC3339: "", Hours: float64(c.hours)})
	case stripeActionDelete:
		return p.deleteRow(c.row)
	default:
		return nil
	}
}

func (p *stripePane) confirmView() string {
	if p.stripeConfirm == nil {
		return ""
	}
	title := "Advance test clock"
	if p.stripeConfirm.action == stripeActionDelete {
		title = "Delete test clock"
	}
	return renderConfirmScreen(p.width, title, "", p.stripeConfirmLines(), p.stripeConfirm.cursor)
}

func (p *stripePane) confirmKeyHints() []string {
	return confirmKeyHints(p.stripeConfirmLines(), p.stripeConfirm.cursor)
}
