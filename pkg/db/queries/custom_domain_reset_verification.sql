-- name: ResetCustomDomainVerification :exec
UPDATE custom_domains
SET verification_status = sqlc.arg(verification_status),
    check_attempts = sqlc.arg(check_attempts),
    verification_error = NULL,
    last_checked_at = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE domain = sqlc.arg(domain);
