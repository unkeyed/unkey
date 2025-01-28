-- name: DeleteRatelimitNamespace :execresult
UPDATE `ratelimit_namespaces`
SET deleted_at = NOW()
WHERE id = sqlc.arg(id);
