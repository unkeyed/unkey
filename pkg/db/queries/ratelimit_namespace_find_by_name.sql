-- name: FindRatelimitNamespaceByName :one
SELECT ratelimit_namespaces.pk, ratelimit_namespaces.id, ratelimit_namespaces.workspace_id, ratelimit_namespaces.project_id, ratelimit_namespaces.name, ratelimit_namespaces.created_at_m, ratelimit_namespaces.updated_at_m, ratelimit_namespaces.deleted_at_m FROM `ratelimit_namespaces`
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id);
