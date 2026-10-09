-- name: LockRoleByIdOrName :one
(
    SELECT r1.id, r1.project_id, r1.name, r1.description, 0 AS lookup_priority
    FROM roles r1
    WHERE r1.workspace_id = sqlc.arg(workspace_id) AND r1.id = sqlc.arg('search')
    FOR UPDATE
)
UNION ALL
(
    SELECT r2.id, r2.project_id, r2.name, r2.description, 1 AS lookup_priority
    FROM roles r2
    WHERE r2.workspace_id = sqlc.arg(workspace_id) AND r2.name = sqlc.arg('search')
    FOR UPDATE
)
ORDER BY lookup_priority
LIMIT 1;
