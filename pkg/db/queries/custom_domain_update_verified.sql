-- name: UpdateCustomDomainVerified :exec
UPDATE custom_domains
SET verification_status = sqlc.arg(verification_status),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
