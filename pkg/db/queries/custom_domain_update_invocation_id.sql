-- name: UpdateCustomDomainInvocationID :exec
UPDATE custom_domains
SET invocation_id = sqlc.narg(invocation_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
