-- name: UpdateRatelimitNamespaceUseGlobalRatelimit :exec
UPDATE ratelimit_namespaces
SET
    use_global_ratelimit = sqlc.arg('use_global_ratelimit'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
