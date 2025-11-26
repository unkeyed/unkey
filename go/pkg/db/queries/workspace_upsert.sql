-- name: UpsertWorkspace :exec
INSERT INTO workspaces (
    id,
    org_id,
    name,
    slug,
    created_at_m,
    tier,
    beta_features,
    features,
    enabled,
    delete_protection
) VALUES (?, ?, ?, ?, ?, ?, ?, '{}', true, false)
ON DUPLICATE KEY UPDATE
    beta_features = VALUES(beta_features),
    name = VALUES(name);
