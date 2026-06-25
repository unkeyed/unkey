-- name: LockKeyForUpdate :one
-- Acquires an exclusive lock on the key row to prevent concurrent modifications.
-- This is used to prevent deadlocks when updating key ratelimits concurrently.
SELECT id FROM `keys`
WHERE id = sqlc.arg(id)
FOR UPDATE;
