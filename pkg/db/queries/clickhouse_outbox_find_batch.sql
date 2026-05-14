-- name: FindClickhouseOutboxBatch :many
-- FindClickhouseOutboxBatch returns the next batch of unprocessed outbox
-- rows for a known set of payload versions. Must be called inside a
-- transaction. FOR UPDATE SKIP LOCKED locks the batch so a second cron tick
-- (if Restate VO serialization ever fails) silently skips them rather than
-- re-processing the same set. The lock is released when the caller commits
-- or rolls back. Ordered by pk so retries see a deterministic row set,
-- which lets CH's block-level deduplication collapse re-inserts after a
-- partial failure.
--
-- The version filter means a drainer never reads a payload it can't
-- decode. Unknown versions stay in the table until a drainer with the
-- matching handler ships.
--
-- deleted_at IS NULL skips rows the drainer already shipped. Marked rows
-- stay in the table for re-processing (clear deleted_at to re-queue) and
-- as an ops audit trail; there's no sweep job today.
SELECT pk, version, workspace_id, event_id, payload, created_at
FROM clickhouse_outbox
WHERE version IN (sqlc.slice('versions'))
  AND deleted_at IS NULL
ORDER BY pk
LIMIT ?
FOR UPDATE SKIP LOCKED;
