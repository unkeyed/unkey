-- name: UpdateCustomDomainOwnership :exec
UPDATE custom_domains
SET ownership_verified = ?, cname_verified = ?, updated_at = ?
WHERE id = ?;
