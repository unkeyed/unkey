-- name: UpdateCustomDomainVerificationError :exec
UPDATE custom_domains
SET verification_error = sqlc.arg(verification_error),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
