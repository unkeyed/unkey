-- name: Read :one
-- Read sums the rows of a name that the calling writer does not count itself.
-- The writer holds its current and previous rows in memory, which is ahead of
-- the replica, so including the replica's copy would count them twice or count
-- an old value. Older writer ids of the same writer are no longer in memory, so
-- they are summed like any other writer's rows.
SELECT COALESCE(SUM(count), 0) AS total
FROM counters
WHERE name = sqlc.arg(name)
    AND writer_id <> sqlc.arg(current_writer_id)
    AND writer_id <> sqlc.arg(previous_writer_id);
