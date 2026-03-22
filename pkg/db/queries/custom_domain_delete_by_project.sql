-- name: DeleteCustomDomainsByProjectId :exec
DELETE FROM custom_domains WHERE project_id = sqlc.arg(project_id);
