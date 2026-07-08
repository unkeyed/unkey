-- name: UpdateCustomDomainCheckAttempt :exec
UPDATE custom_domains
SET check_attempts = sqlc.arg(check_attempts),
    last_checked_at = sqlc.arg(last_checked_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
