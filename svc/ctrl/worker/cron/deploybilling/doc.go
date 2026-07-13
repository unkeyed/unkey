// Package deploybilling runs the hourly Deploy usage push and month-end close.
//
// Each tick reads month-to-date totals from ClickHouse and pushes absolute
// values to Stripe ("last" meters). Retries converge on the same number.
// Month-end close pushes the final period total and finalizes renewal drafts.
// The invoice.created webhook closes one workspace per invoice
// (HandleCloseWorkspace); the 00:30 UTC backup cron runs a fleet sweep
// (HandleClose). Keyed by billing period "YYYY-MM" for the sweep and by
// workspace id for per-invoice close. No-op without ClickHouse or Stripe.
package deploybilling
