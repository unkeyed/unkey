-- name: DeleteDrainedBackofficeAuditOutbox :execrows
-- DeleteDrainedBackofficeAuditOutbox hard-deletes a bounded batch of back
-- office audit outbox rows that the back office app already copied to
-- ClickHouse (drained_at stamped) before the retention cutoff, and returns
-- the number of rows deleted so the caller can loop until the table is
-- drained of expired rows.
--
-- The back office app itself never deletes: its MySQL user has no DELETE
-- grant, and it stamps drained_at instead so ops can re-queue a row (clear
-- drained_at) or audit recent exports. This sweep, run by the same daily
-- cron as the clickhouse_outbox sweep, reclaims the space once the window
-- has passed.
--
-- drained_at < cutoff implicitly skips pending and parked rows: drained_at
-- IS NULL never satisfies the comparison. The drainer_pending_idx
-- (drained_at, pk) leading on drained_at turns this into a range seek.
--
-- LIMIT bounds each DELETE so locks stay short and replication lag stays
-- bounded; the cron loops until a batch deletes fewer than the limit.
DELETE FROM backoffice_audit_outbox
WHERE drained_at < sqlc.arg('cutoff')
LIMIT ?;
