-- name: Read :one
-- Read sums every row of a name except the writer's current and previous rows,
-- which the writer holds in memory. Those come from memory, including changes
-- that have not been persisted yet. The writer's older rows are no longer held
-- in memory, so they count like any other writer's.
-- SUM returns NULL when no row qualifies; treat that as zero.
SELECT COALESCE(SUM(count), 0) AS total
FROM counters
WHERE name = sqlc.arg(name)
    AND writer_id <> sqlc.arg(current_writer_id)
    AND writer_id <> sqlc.arg(previous_writer_id);
