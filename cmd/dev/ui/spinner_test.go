package ui

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

// Reproduces the routing bug: when Seed and Stripe both load at once, Seed is
// earlier in dispatch order. Stripe's tick must still reach Stripe, not be
// swallowed by Seed, so the Stripe spinner keeps animating.
func TestStripeSpinnerNotStolenBySeed(t *testing.T) {
	chdirTemp(t)
	m := newAppModel()
	mm, _ := m.Update(app.WindowSizeMsg{Width: 120, Height: 30})
	m = mm.(appModel)

	seed := m.panes[tabSeed].(*seedPane)
	stripe := m.panes[tabStripe].(*stripePane)
	seed.loading = true
	stripe.loading = true

	stripeBefore := stripe.spinner.View()
	seedBefore := seed.spinner.View()

	// Deliver a tick that belongs to Stripe while Seed is also loading.
	mm, _ = m.Update(app.SpinnerTickMsg{ID: stripe.spinner.ID()})
	m = mm.(appModel)

	require.NotEqual(t, stripeBefore, stripe.spinner.View(), "stripe spinner advances on its own tick")
	require.Equal(t, seedBefore, seed.spinner.View(), "seed spinner does not consume stripe's tick")
}
