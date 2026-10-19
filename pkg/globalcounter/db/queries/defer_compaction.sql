-- name: DeferCompaction :exec
-- DeferCompaction gives the given rows of one name a compaction level and moves
-- their compaction_after to the database time plus delay_ms, so compaction passes
-- stop selecting rows that they cannot merge: a name with a single compactable
-- row, or a sum that overflows.
UPDATE counters
SET compaction_level = sqlc.arg(compaction_level),
    compaction_after = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + sqlc.arg(delay_ms)
WHERE name = sqlc.arg(name)
    AND writer_id IN (sqlc.slice('writer_ids'));
