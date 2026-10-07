-- name: UpsertWorkspace :exec
-- UpsertWorkspace seeds local workspaces while preserving fields that local tooling does not manage.
-- New rows receive a caller-generated Kubernetes namespace; existing rows retain their namespace.
INSERT INTO workspaces (
    id,
    org_id,
    name,
    slug,
    created_at_m,
    beta_features,
    k8s_namespace,
    enabled,
    delete_protection
) VALUES (?, ?, ?, ?, ?, ?, ?, true, false)
ON DUPLICATE KEY UPDATE
    beta_features = VALUES(beta_features),
    name = VALUES(name);
