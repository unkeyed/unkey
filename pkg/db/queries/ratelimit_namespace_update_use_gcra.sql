-- name: UpdateRatelimitNamespaceUseGcra :exec
UPDATE ratelimit_namespaces
SET
    use_gcra = sqlc.arg('use_gcra'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
