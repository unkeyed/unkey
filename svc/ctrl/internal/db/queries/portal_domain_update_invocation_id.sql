-- name: UpdatePortalDomainInvocationID :exec
-- Records the verification workflow invocation driving this domain; NULL clears it.
UPDATE portal_domains
SET invocation_id = sqlc.narg(invocation_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
