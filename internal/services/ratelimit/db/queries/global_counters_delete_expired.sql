-- name: GlobalCountersDeleteExpired :execrows
-- GlobalCountersDeleteExpired removes a bounded batch of rows whose grace
-- period has passed and returns the number of rows deleted so the caller can
-- loop until the table is drained of expired rows. Called by an external
-- Restate cron, not the ratelimit service itself.
--
-- LIMIT is required, not an optimization: PlanetScale rejects any single DML
-- statement that would affect more than 100,000 rows, so an unbounded DELETE
-- fails outright once a backlog builds. Bounding each batch below that ceiling
-- also keeps row locks short and replication lag contained.
--
-- expires_at_idx makes this a range seek rather than a full scan.
DELETE FROM ratelimit_global_counters
WHERE expires_at < sqlc.arg("cutoff")
LIMIT ?;
