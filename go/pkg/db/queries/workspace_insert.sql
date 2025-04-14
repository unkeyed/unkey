-- name: InsertWorkspace :exec
INSERT INTO `workspaces` (
    id,
    org_id,
    name,
    created_at_m,
    tier,
    beta_features,
    features,
    enabled,
    delete_protection
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(org_id),
    sqlc.arg(name),
     sqlc.arg(created_at),
    'Free',
    '{}',
    '{}',
    true,
    true
);
