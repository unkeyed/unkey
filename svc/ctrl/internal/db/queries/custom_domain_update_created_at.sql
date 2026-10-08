-- name: UpdateCustomDomainCreatedAt :exec
-- UpdateCustomDomainCreatedAt backdates a custom domain the service created, so
-- a test can control its age. Only tests use this query.
UPDATE `custom_domains`
SET created_at = sqlc.arg(created_at)
WHERE id = sqlc.arg(id);
