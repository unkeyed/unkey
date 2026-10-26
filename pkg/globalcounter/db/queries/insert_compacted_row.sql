-- name: InsertCompactedRow :exec
-- InsertCompactedRow stores merged rows as one new row. It uses a new writer id
-- instead of updating one of the sources, so no writer can ever write it and its
-- revision never matters. The caller deletes the sources in the same transaction,
-- so the total of the name never changes, not even for a moment.
INSERT INTO counters (
    name,
    writer_id,
    count,
    revision,
    delete_after,
    compaction_after,
    compaction_level
) VALUES (
    sqlc.arg(name),
    sqlc.arg(writer_id),
    sqlc.arg(count),
    1,
    sqlc.narg(delete_after),
    CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + sqlc.arg(compact_delay_ms),
    sqlc.arg(compaction_level)
);
