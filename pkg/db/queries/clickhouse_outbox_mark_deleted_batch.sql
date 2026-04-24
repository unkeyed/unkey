-- name: MarkClickhouseOutboxBatchDeleted :exec
-- MarkClickhouseOutboxBatchDeleted soft-deletes a set of pks after their CH
-- insert is confirmed. Called inside the same transaction that selected
-- them, so the row locks held by FOR UPDATE SKIP LOCKED are released as
-- part of commit. A crash between the CH insert and this UPDATE leaves the
-- rows with deleted_at IS NULL; the next batch picks them up and CH's
-- non_replicated_deduplication_window collapses the identical re-insert
-- into a noop.
--
-- We mark instead of hard-delete so ops can re-queue events (clear
-- deleted_at) without re-reading the original payload from somewhere else,
-- and so the table doubles as an audit trail of what was exported. There's
-- no sweep job today; the table grows monotonically.
UPDATE clickhouse_outbox
SET deleted_at = sqlc.arg(deleted_at)
WHERE pk IN (sqlc.slice('pks'))
  AND deleted_at IS NULL;
