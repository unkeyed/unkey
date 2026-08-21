-- name: InsertClickhouseOutbox :exec
-- transactional-batch-statement
-- InsertClickhouseOutbox enqueues one event for ClickHouse export. Called
-- from the same MySQL transaction as the underlying mutation, so durability
-- is exactly the durability of the mutation: if the mutation commits, the
-- outbox row commits.
--
-- version namespaces the payload schema (e.g. "audit_log.v1"). The drainer
-- filters by versions it knows, so writing a new version without a matching
-- drainer leaves rows queued safely.
INSERT INTO `clickhouse_outbox` (
    version,
    workspace_id,
    event_id,
    payload,
    created_at
) VALUES (
    sqlc.arg(version),
    sqlc.arg(workspace_id),
    sqlc.arg(event_id),
    CAST(sqlc.arg(payload) AS JSON),
    sqlc.arg(created_at)
);

-- name: InsertClickhouseOutboxForCreditUpdate :exec
-- transactional-batch-statement
-- This statement must immediately follow the guarded credit UPDATE. With
-- clientFoundRows enabled on the batch pool, ROW_COUNT() distinguishes a
-- missing key while LAST_INSERT_ID() distinguishes valid and rejected states.
INSERT INTO `clickhouse_outbox` (
    version,
    workspace_id,
    event_id,
    payload,
    created_at
)
SELECT
    sqlc.arg(version),
    sqlc.arg(workspace_id),
    sqlc.arg(event_id),
    JSON_SET(
        CAST(sqlc.arg(payload) AS JSON),
        '$.description',
        CONCAT(
            'Updated Key ', sqlc.arg(key_id), ', set remaining to ',
            IF(k.remaining_requests IS NULL, 'unlimited', CAST(k.remaining_requests AS CHAR)),
            '.'
        )
    ),
    sqlc.arg(created_at)
FROM `keys` k
WHERE k.id = sqlc.arg(key_id)
  AND ROW_COUNT() = 1
  AND LAST_INSERT_ID() <= 9223372036854775807;
