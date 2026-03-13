-- name: InsertWorkspace :exec
INSERT INTO `workspaces` (
    id,
    org_id,
    name,
    slug,
    created_at_m,
    tier,
    beta_features,
    features,
    enabled,
    delete_protection,
    k8s_namespace
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(org_id),
    sqlc.arg(name),
    sqlc.arg(slug),
    sqlc.arg(created_at),
    'Free',
    '{}',
    '{}',
    true,
    true,
    sqlc.arg(k8s_namespace)
);
