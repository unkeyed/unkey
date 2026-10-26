-- name: DeleteExpired :execrows
-- DeleteExpired removes rows whose caller-defined lifetime has ended. The LIMIT
-- keeps each statement short, so a backlog after an outage cannot hold locks on
-- the primary for long. The caller picks a cutoff that leaves room for writes
-- still in flight, because deleting a row also forgets its revision. Rows with a
-- NULL delete_after are never deleted.
DELETE FROM counters
WHERE delete_after <= sqlc.arg(cutoff_ms)
ORDER BY delete_after
LIMIT ?;
