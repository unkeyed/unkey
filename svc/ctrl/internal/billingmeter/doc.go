// Package billingmeter reports workspace usage to a billing provider's metering
// API. A [Pusher] takes one workspace's month-to-date totals and records them
// against the provider's meters; the deploy billing cron computes the totals
// and calls a Pusher each tick.
//
// The Stripe implementation posts billing meter events whose meters aggregate
// with formula "last", so sending the absolute month-to-date value overwrites
// (rather than increments) the period's quantity. Re-runs, overlapping ticks,
// and replays therefore converge on the same billed number.
package billingmeter
