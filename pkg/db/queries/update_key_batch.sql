-- name: LockWorkspaceForUpdateKey :one
-- transactional-batch-statement
SELECT id
FROM workspaces
WHERE id = sqlc.arg(workspace_id)
    AND id = sqlc.arg(workspace_id_check)
FOR UPDATE;

-- name: InsertDefaultProjectForUpdateKey :exec
-- transactional-batch-statement
-- no-bulk-insert
INSERT INTO projects (
    id,
    workspace_id,
    name,
    slug,
    delete_protection,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    'Default',
    'default',
    true,
    sqlc.arg(created_at),
    NULL
)
ON DUPLICATE KEY UPDATE id = projects.id;

-- name: InsertIdentityForUpdateKey :exec
-- transactional-batch-statement
-- no-bulk-insert
INSERT INTO identities (
    id,
    external_id,
    workspace_id,
    project_id,
    environment,
    created_at,
    meta
)
SELECT
    sqlc.arg(id),
    sqlc.arg(external_id),
    sqlc.arg(workspace_id),
    p.id,
    'default',
    sqlc.arg(created_at),
    JSON_OBJECT()
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND BINARY p.slug = 'default'
ON DUPLICATE KEY UPDATE id = identities.id;

-- name: InsertPermissionForUpdateKey :exec
-- transactional-batch-statement
-- no-bulk-insert
INSERT INTO permissions (
    id,
    workspace_id,
    project_id,
    name,
    slug,
    description,
    created_at_m
)
SELECT
    sqlc.arg(permission_id),
    sqlc.arg(workspace_id),
    p.id,
    sqlc.arg(name),
    sqlc.arg(slug),
    sqlc.narg(description),
    sqlc.arg(created_at_m)
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND BINARY p.slug = 'default'
ON DUPLICATE KEY UPDATE id = permissions.id;

-- name: DeleteKeyPermissionsForUpdateKey :exec
-- transactional-batch-statement
DELETE kp
FROM keys_permissions kp
JOIN `keys` k ON k.id = kp.key_id
WHERE k.id = sqlc.arg(key_id)
    AND k.workspace_id = sqlc.arg(workspace_id);

-- name: DeleteKeyRolesForUpdateKey :exec
-- transactional-batch-statement
DELETE kr
FROM keys_roles kr
JOIN `keys` k ON k.id = kr.key_id
WHERE k.id = sqlc.arg(key_id)
    AND k.workspace_id = sqlc.arg(workspace_id);

-- name: DeleteKeyPermissionsAndRolesForUpdateKey :exec
-- transactional-batch-statement
DELETE kp, kr
FROM `keys` k
LEFT JOIN keys_permissions kp ON k.id = kp.key_id
LEFT JOIN keys_roles kr ON k.id = kr.key_id
WHERE k.id = sqlc.arg(key_id)
    AND k.workspace_id = sqlc.arg(workspace_id);

-- name: InsertKeyPermissionBySlugForUpdateKey :exec
-- transactional-batch-statement
-- no-bulk-insert
INSERT INTO keys_permissions (
    key_id,
    permission_id,
    workspace_id,
    created_at_m,
    updated_at_m
)
SELECT
    sqlc.arg(key_id),
    p.id,
    sqlc.arg(workspace_id),
    sqlc.arg(created_at_m),
    sqlc.narg(updated_at_m)
FROM permissions p
WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND p.slug = sqlc.arg(permission_slug)
ON DUPLICATE KEY UPDATE updated_at_m = VALUES(updated_at_m);

-- name: DeleteKeyRatelimitsForUpdateKey :exec
-- transactional-batch-statement
DELETE rl
FROM ratelimits rl
JOIN `keys` k ON k.id = rl.key_id
WHERE rl.key_id = sqlc.arg(key_id)
    AND k.workspace_id = sqlc.arg(workspace_id);

-- name: InsertClickhouseOutboxForPermissionUpdateKey :exec
-- transactional-batch-statement
-- no-bulk-insert
INSERT INTO clickhouse_outbox (
    version,
    workspace_id,
    event_id,
    payload,
    created_at
)
SELECT
    sqlc.arg(version),
    sqlc.arg(workspace_id),
    sqlc.arg(event_id),
    CAST(sqlc.arg(payload) AS JSON),
    sqlc.arg(created_at)
WHERE EXISTS (
    SELECT 1
    FROM permissions p
    WHERE p.id = sqlc.arg(permission_id)
);
