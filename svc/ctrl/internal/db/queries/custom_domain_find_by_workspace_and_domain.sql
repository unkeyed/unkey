-- name: FindCustomDomainByWorkspaceAndDomain :one
SELECT
    id,
    project_id,
    app_id,
    environment_id,
    domain,
    invocation_id
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id) AND domain = sqlc.arg(domain);
