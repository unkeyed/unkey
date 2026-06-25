-- name: LockIdentityForUpdate :one
-- Acquires an exclusive lock on the identity row to prevent concurrent modifications.
-- This should be called at the start of a transaction before modifying identity-related data.
SELECT id FROM identities
WHERE id = sqlc.arg(id)
FOR UPDATE;
