-- name: ListRatelimitNamespaceOwnershipByWorkspace :many
-- Resolves canonical analytics grants to namespace IDs owned by one workspace.
-- Soft-deleted namespaces remain present because their historical ClickHouse
-- rows must stay queryable, and the unpaginated result prevents scope loss.
SELECT id, project_id
FROM ratelimit_namespaces
WHERE workspace_id = sqlc.arg(workspace_id);
