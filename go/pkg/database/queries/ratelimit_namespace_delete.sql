-- name: DeleteRatelimitNamespace :execresult
UPDATE `ratelimit_namespaces`
SET deleted_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
