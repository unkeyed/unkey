-- name: ListExpiredRows :many
-- ListExpiredRows returns rows that expired before the cutoff, after a cursor,
-- in the order of the expires_at_ns index, so a page scans at most LIMIT index
-- entries. The caller must still check every row of a name before deleting it.
SELECT name, expires_at_ns
FROM ratelimit_gcra
WHERE expires_at_ns < sqlc.arg(cutoff_ns)
    AND (
        expires_at_ns > sqlc.arg(after_expires_at_ns)
        OR (expires_at_ns = sqlc.arg(after_expires_at_ns) AND name > sqlc.arg(after_name))
    )
ORDER BY expires_at_ns, name
LIMIT ?;
