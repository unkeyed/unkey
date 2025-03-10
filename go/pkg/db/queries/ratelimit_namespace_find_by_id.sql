-- name: FindRatelimitNamespaceByID :one
SELECT * FROM `ratelimit_namespaces`
WHERE id = sqlc.arg(id);
