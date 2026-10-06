-- name: DeletePortalDomainByID :exec
-- Unscoped by workspace: callers resolve ownership before deleting.
DELETE FROM portal_domains WHERE id = sqlc.arg(id);
