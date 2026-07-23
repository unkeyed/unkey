-- name: FindRatelimitNamespacesByIDs :many
-- FindRatelimitNamespacesByIDs scopes matches to one workspace and intentionally
-- includes soft-deleted namespaces so historical analytics remain authorized.
SELECT id
FROM ratelimit_namespaces
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id IN (sqlc.slice(namespace_ids));
