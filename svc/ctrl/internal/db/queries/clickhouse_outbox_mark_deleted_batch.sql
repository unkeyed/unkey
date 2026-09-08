-- name: MarkClickhouseOutboxBatchDeleted :exec
-- MarkClickhouseOutboxBatchDeleted soft-deletes a set of pks after their CH
-- insert is confirmed. It runs as its own short autocommit statement, not in
-- the transaction that selected the rows, so no MySQL lock spans the
-- ClickHouse insert. The pk list is exactly the set the drainer read and
-- ClickHouse acknowledged; rows inserted concurrently are never in it. The
-- deleted_at IS NULL guard keeps a replayed or overlapping mark from
-- restamping rows an earlier attempt already marked.
--
-- A crash between the CH insert and this UPDATE leaves the rows with
-- deleted_at IS NULL. The next batch picks them up again, which can create
-- duplicate ClickHouse rows under the at-least-once delivery contract.
--
-- We mark instead of hard-delete so ops can re-queue events (clear
-- deleted_at) without re-reading the original payload from somewhere else,
-- and so the table doubles as an audit trail of what was exported.
-- RunAuditLogOutboxCleanup sweeps marked rows after the retention window.
UPDATE clickhouse_outbox
SET deleted_at = sqlc.arg(deleted_at)
WHERE pk IN (sqlc.slice('pks'))
  AND deleted_at IS NULL;
