-- name: FindCustomDomainByWorkspaceAndDomain :one
SELECT * FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id) AND domain = sqlc.arg(domain);
