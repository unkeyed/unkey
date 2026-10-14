-- name: LockRatelimitNamespaceByIdOrName :one
(
    SELECT n1.id, n1.project_id, n1.name, n1.created_at_m, n1.updated_at_m, 0 AS lookup_priority
    FROM ratelimit_namespaces n1
    WHERE n1.workspace_id = sqlc.arg(workspace_id) AND n1.id = sqlc.arg('search') AND n1.deleted_at_m IS NULL
    FOR UPDATE
)
UNION ALL
(
    SELECT n2.id, n2.project_id, n2.name, n2.created_at_m, n2.updated_at_m, 1 AS lookup_priority
    FROM ratelimit_namespaces n2
    WHERE n2.workspace_id = sqlc.arg(workspace_id) AND n2.name = sqlc.arg('search') AND n2.deleted_at_m IS NULL
    FOR UPDATE
)
ORDER BY lookup_priority
LIMIT 1;
