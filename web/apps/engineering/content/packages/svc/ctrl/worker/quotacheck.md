---
title: quotacheck
description: "implements a Restate workflow for monitoring workspace quota usage"
---

Package quotacheck implements a Restate workflow for monitoring workspace quota usage.

This service checks all workspaces for exceeded quotas and sends Slack notifications. It uses Restate's virtual object state to deduplicate notifications - each workspace is only notified once per billing period when they first exceed their quota.

The service is keyed by billing period (e.g., "2026-01") which allows:

  - Tracking which workspaces have been notified this month
  - Resetting notification state automatically each billing period
  - Sending end-of-month summaries

Self-scheduling pattern:

  - After each RunCheck, the service schedules the next run 24 hours later
  - Uses idempotency keys (e.g., "quota-check-2026-01-15") to prevent duplicate runs
  - On the last day of the month, schedules SendMonthlySummary for an overview
  - When month changes, creates a new VO with fresh state for the new billing period

Performance optimizations:

  - Bulk queries to ClickHouse (2 queries total instead of 2N)
  - Single MySQL query to get all workspace quotas
  - Restate state prevents duplicate Slack notifications

Bootstrap: To start the daily check cycle, call RunCheck once with the current billing period:

	POST /QuotaCheckService/2026-01/RunCheck
	{"slack_webhook_url": "https://hooks.slack.com/..."}

## Constants

batchSize is the number of workspace IDs to fetch from the database in a single query. This balances between minimizing round trips and keeping queries efficient.
```go
const batchSize = 100
```

followUpInterval is the minimum time between follow-up notifications. First notification is sent immediately, subsequent ones are sent weekly. We use 6 days 20 hours instead of exactly 7 days to account for timing drift in the daily scheduled job (e.g., 16:03 one week vs 16:00 the next).
```go
const followUpInterval = 6*24*time.Hour + 20*time.Hour
```

minUsageThreshold is the minimum usage to consider for quota checks. Workspaces below this threshold are skipped since the minimum paid plan starts at 150k.
```go
const minUsageThreshold = 150_000
```

```go
const stateKeyNotifiedWorkspaces = "notified_workspaces"
```


## Variables

```go
var _ hydrav1.QuotaCheckServiceServer = (*Service)(nil)
```


## Functions


## Types

### type Config

```go
type Config struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	// Heartbeat sends health signals after successful quota check runs.
	// Must not be nil - use healthcheck.NewNoop() if monitoring is not needed.
	Heartbeat healthcheck.Heartbeat
	// SlackWebhookURL is the webhook URL for sending quota exceeded notifications.
	// If empty, no Slack notifications are sent.
	SlackWebhookURL string
}
```

Config holds the configuration for the quota check service.

### type Service

```go
type Service struct {
	hydrav1.UnimplementedQuotaCheckServiceServer
	db              db.Database
	clickhouse      clickhouse.ClickHouse
	heartbeat       healthcheck.Heartbeat
	slackWebhookURL string
}
```

Service implements the QuotaCheckService Restate virtual object.

#### func New

```go
func New(cfg Config) (*Service, error)
```

New creates a new quota check service.

#### func (Service) RunCheck

```go
func (s *Service) RunCheck(
	ctx restate.ObjectContext,
	req *hydrav1.RunCheckRequest,
) (*hydrav1.RunCheckResponse, error)
```

RunCheck queries all workspace usage and sends Slack notifications for newly exceeded quotas. This handler is intended to be called on a schedule via GitHub Actions.

