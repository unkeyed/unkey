-- name: UpdatePortalDomainCheckAttempt :exec
-- Records one DNS check so callers can bound retries and show when the domain
-- was last checked.
UPDATE portal_domains
SET check_attempts = sqlc.arg(check_attempts),
    last_checked_at = sqlc.arg(last_checked_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
