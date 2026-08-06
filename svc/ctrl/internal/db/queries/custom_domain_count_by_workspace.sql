-- name: CountCustomDomainsByWorkspace :one
-- Covered by unique_domain_workspace_idx, which leads on workspace_id.
SELECT COUNT(*)
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id);
