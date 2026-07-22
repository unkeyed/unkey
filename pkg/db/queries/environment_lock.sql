-- name: LockEnvironmentForUpdate :one
-- Acquires an exclusive lock on the environment row to prevent concurrent modifications.
-- This serializes region reconciliation, which reads the current set then replaces it.
SELECT id FROM environments
WHERE id = sqlc.arg(id)
FOR UPDATE;
