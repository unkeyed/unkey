-- name: ReadPeers :one
-- ReadPeers combines every row of a name except the calling writer's live row,
-- which the writer holds in memory ahead of the replica. Older writer ids of the
-- same process are not in memory any more and count like any other writer.
SELECT
    COALESCE(SUM(spent_ns), 0) AS spent_ns,
    COALESCE(MAX(fresh_start_ns), 0) AS fresh_start_ns
FROM ratelimit_gcra
WHERE name = sqlc.arg(name)
    AND writer_id <> sqlc.arg(writer_id);
