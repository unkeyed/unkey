-- name: InsertCompactedRow :exec
-- InsertCompactedRow stores the sum of rows that compaction merged as one new row
-- under a writer id nobody writes. Callers delete the merged rows in the same
-- transaction. The row's compaction_after is the database time plus compact_delay_ms,
-- and its compaction level is the lowest level of the rows it replaces.
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
