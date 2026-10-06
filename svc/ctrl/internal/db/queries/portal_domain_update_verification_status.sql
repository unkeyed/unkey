-- name: UpdatePortalDomainVerificationStatus :exec
-- Moves a domain between verification states without touching the error or
-- attempt counters.
UPDATE portal_domains
SET verification_status = sqlc.arg(verification_status),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
