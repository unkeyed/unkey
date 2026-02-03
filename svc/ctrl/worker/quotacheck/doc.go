// Package quotacheck implements a Restate workflow for monitoring workspace quota usage.
//
// This service checks all workspaces for exceeded quotas and sends Slack notifications.
// It uses Restate's virtual object state to deduplicate notifications - each workspace
// is only notified once per billing period when they first exceed their quota.
//
// The service is keyed by billing period (e.g., "2026-01") which allows:
//   - Tracking which workspaces have been notified this month
//   - Resetting notification state automatically each billing period
//   - Sending end-of-month summaries
//
// Self-scheduling pattern:
//   - After each RunCheck, the service schedules the next run 24 hours later
//   - Uses idempotency keys (e.g., "quota-check-2026-01-15") to prevent duplicate runs
//   - On the last day of the month, schedules SendMonthlySummary for an overview
//   - When month changes, creates a new VO with fresh state for the new billing period
//
// Performance optimizations:
//   - Bulk queries to ClickHouse (2 queries total instead of 2N)
//   - Single MySQL query to get all workspace quotas
//   - Restate state prevents duplicate Slack notifications
//
// Bootstrap:
// To start the daily check cycle, call RunCheck once with the current billing period:
//
//	POST /QuotaCheckService/2026-01/RunCheck
//	{"slack_webhook_url": "https://hooks.slack.com/..."}
package quotacheck
