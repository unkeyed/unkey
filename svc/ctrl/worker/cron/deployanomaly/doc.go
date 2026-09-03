// Package deployanomaly classifies closed Deploy metric windows against a
// trailing baseline. It contains no IO or durable state; the cron orchestrator
// supplies ClickHouse aggregates and the per-app virtual object supplies the
// previous-candidate state used to confirm error anomalies.
package deployanomaly
