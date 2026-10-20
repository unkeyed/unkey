-- name: UpdateRatelimitNamespaceUseGcra :exec
UPDATE `ratelimit_namespaces`
SET
    use_gcra = sqlc.arg(use_gcra),
    updated_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(id);
