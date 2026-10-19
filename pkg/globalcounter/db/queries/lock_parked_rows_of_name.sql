-- name: LockParkedRowsOfName :many
-- LockParkedRowsOfName reads up to a limit of the rows of one name that compaction
-- created or moved ahead, and that are not due yet, and locks them until the
-- transaction ends. These are the rows with a compaction level above zero and the
-- rows whose writer id starts with the given prefix. No writer writes them, so
-- they are final even though compaction waits to look at them. A pass that looks
-- at the name because of a ready row merges them in too. A row another compactor
-- holds is skipped.
SELECT writer_id, count, delete_after, compaction_level
FROM counters
WHERE name = sqlc.arg(name)
    AND compaction_after > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
    AND (compaction_level > 0 OR LOCATE(sqlc.arg(compacted_prefix), writer_id) = 1)
ORDER BY compaction_after
LIMIT ?
FOR UPDATE SKIP LOCKED;
