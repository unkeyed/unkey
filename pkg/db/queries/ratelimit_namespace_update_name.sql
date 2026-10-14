-- name: UpdateRatelimitNamespaceName :exec
UPDATE ratelimit_namespaces
SET
    name = sqlc.arg('name'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
