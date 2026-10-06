-- name: DeletePortalDomainByID :exec
DELETE FROM portal_domains WHERE id = sqlc.arg(id);
