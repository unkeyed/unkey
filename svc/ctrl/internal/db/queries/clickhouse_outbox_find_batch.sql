-- name: FindClickhouseOutboxBatch :many
-- FindClickhouseOutboxBatch returns the next batch of unprocessed outbox
-- rows for known payload versions, ordered by pk. Use an autocommit read:
-- locking the pending index across a ClickHouse call blocks API inserts.
-- Restate serializes the exporter key, but overlapping canceled attempts
-- can still deliver duplicates under the at-least-once contract.
SELECT pk, version, workspace_id, event_id, payload, created_at
FROM clickhouse_outbox
WHERE version IN (sqlc.slice('versions'))
  AND deleted_at IS NULL
ORDER BY pk
LIMIT ?;
