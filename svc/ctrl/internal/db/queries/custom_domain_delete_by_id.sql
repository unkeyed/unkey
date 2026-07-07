-- name: DeleteCustomDomainByID :exec
DELETE FROM custom_domains WHERE id = sqlc.arg(id);
