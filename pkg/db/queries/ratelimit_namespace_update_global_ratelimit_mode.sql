-- name: UpdateRatelimitNamespaceGlobalRatelimitMode :exec
UPDATE ratelimit_namespaces
SET
    global_ratelimit_mode = sqlc.arg('global_ratelimit_mode'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
