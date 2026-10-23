-- name: LockCompactableRowsOfName :many
-- LockCompactableRowsOfName reads up to a limit of one name's rows whose
-- compaction_after has passed by the database clock, oldest first, and locks them
-- until the transaction ends. A row another compactor holds is skipped, so
-- compactors working at the same time take different rows instead of waiting for
-- each other. The limit counts rows, so one name with a long history cannot make
-- a transaction unbounded.
SELECT writer_id, count, delete_after, compaction_level
FROM counters
WHERE name = sqlc.arg(name)
    AND compaction_after <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
ORDER BY compaction_after
LIMIT ?
FOR UPDATE SKIP LOCKED;
