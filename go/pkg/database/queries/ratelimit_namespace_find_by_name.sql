-- name: FindRatelimitNamespaceByName :one
SELECT * FROM `ratelimit_namespaces`
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id);
