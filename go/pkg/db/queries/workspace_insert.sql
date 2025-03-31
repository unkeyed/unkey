-- name: InsertWorkspace :exec
INSERT INTO `workspaces` (
    id,
    tenant_id,
    name,
    created_at_m,
    plan,
    beta_features,
    features,
    enabled,
    delete_protection
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(tenant_id),
    sqlc.arg(name),
     sqlc.arg(created_at),
    'free',
    '{}',
    '{}',
    true,
    true
);
