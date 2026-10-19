-- name: ListCompactableNames :many
-- ListCompactableNames returns the names of the rows whose compaction_after has
-- passed by the database clock, oldest row first. A name appears once for each
-- of its rows, so the caller removes duplicates. The read takes no locks.
SELECT name
FROM counters
WHERE compaction_after <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
ORDER BY compaction_after
LIMIT ?;
