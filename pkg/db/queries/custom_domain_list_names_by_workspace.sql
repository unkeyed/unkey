-- name: ListCustomDomainNamesByWorkspace :many
-- Covered by unique_domain_workspace_idx, which leads on workspace_id.
SELECT domain
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id);
