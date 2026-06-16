// Package deploybilling implements the CronService.RunDeployBillingPush
// handler: the hourly job that reports month-to-date Deploy usage to the
// billing provider.
//
// Each tick computes the running month-to-date total for the four Deploy
// meters (CPU, memory, egress, disk) from Heimdall checkpoint data in
// ClickHouse, then hands each billable workspace's total to a
// billingmeter.Pusher. The pusher sets (not increments) the period quantity,
// so sending the absolute month-to-date value hourly is idempotent: a retry,
// an overlapping tick, or a replay all converge on the same number. There are
// no per-event deltas to dedupe and no end-of-month timing window; the last
// value the provider received before invoice finalize is the one it bills.
//
// The handler is keyed by billing period "YYYY-MM" so concurrent triggers for
// the same month serialize while different months stay independent. It is a
// no-op when ClickHouse or the billing provider is not configured.
package deploybilling
