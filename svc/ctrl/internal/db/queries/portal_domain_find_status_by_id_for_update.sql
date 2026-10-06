-- name: FindPortalDomainStatusByIdForUpdate :one
-- Locks the row so a delete decides route ownership on the status it commits
-- against, not one a concurrent revoke or verification already changed.
SELECT verification_status
FROM portal_domains
WHERE id = sqlc.arg(id)
FOR UPDATE;
