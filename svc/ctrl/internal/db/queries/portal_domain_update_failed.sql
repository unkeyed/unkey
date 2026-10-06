-- name: UpdatePortalDomainFailed :exec
-- Marks verification as given up and keeps the reason for the tenant to read.
UPDATE portal_domains
SET verification_status = sqlc.arg(verification_status),
    verification_error = sqlc.arg(verification_error),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
