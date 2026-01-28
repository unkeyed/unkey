-- name: UpdateCustomDomainFailed :exec
UPDATE custom_domains
SET verification_status = sqlc.arg(verification_status),
    verification_error = sqlc.arg(verification_error),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
