-- name: DeleteClickhouseOutboxBatch :exec
-- DeleteClickhouseOutboxBatch removes a set of pks after their CH insert
-- is confirmed. Called inside the same transaction that selected them, so
-- the row locks held by FOR UPDATE SKIP LOCKED are released as part of
-- commit. A crash between the CH insert and this DELETE leaves the rows
-- in place; the next batch picks them up and CH's
-- non_replicated_deduplication_window collapses the identical re-insert
-- into a noop.
DELETE FROM clickhouse_outbox WHERE pk IN (sqlc.slice('pks'));
