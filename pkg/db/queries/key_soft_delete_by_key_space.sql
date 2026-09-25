-- name: SoftDeleteKeysByKeySpaceID :execrows
-- SoftDeleteKeysByKeySpaceID tombstones a bounded batch of live keys in one
-- keyspace and returns the number of rows affected so the caller can loop
-- until the keyspace is drained.
--
-- LIMIT is required, not an optimization: PlanetScale rejects any single DML
-- statement that would affect more than 100,000 rows, so an unbounded UPDATE
-- fails outright on a large keyspace. Bounding each batch also keeps row locks
-- short.
--
-- deleted_at_m IS NULL both preserves the timestamp on keys deleted earlier
-- and shrinks the candidate set every batch, so the caller's loop terminates.
-- key_auth_id_deleted_at_idx makes this a range seek rather than a full scan.
UPDATE `keys`
SET deleted_at_m = sqlc.arg(now)
WHERE key_auth_id = sqlc.arg(key_space_id)
  AND deleted_at_m IS NULL
LIMIT ?;
