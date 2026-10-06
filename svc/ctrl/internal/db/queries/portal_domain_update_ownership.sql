-- name: UpdatePortalDomainOwnership :exec
-- Stores the TXT (ownership) and CNAME (routing) results of the latest check.
UPDATE portal_domains
SET ownership_verified = sqlc.arg(ownership_verified),
    cname_verified = sqlc.arg(cname_verified),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
