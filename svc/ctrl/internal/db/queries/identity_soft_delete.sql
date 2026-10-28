-- name: SoftDeleteIdentity :exec
-- Soft-deletes an identity the way the API's identity delete does. Used only by
-- tests that need a soft-deleted identity.
UPDATE identities SET deleted = true, updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id);
