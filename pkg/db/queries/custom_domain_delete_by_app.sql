-- name: DeleteCustomDomainsByAppId :exec
DELETE FROM custom_domains WHERE app_id = sqlc.arg(app_id);
