-- name: ListOrphanedDeploymentTopologiesByRegion :many
-- A missing deployment must not erase the region's removal obligation.
-- An absent workspace leaves the namespace empty so Krane searches by deployment ID.
SELECT dt.pk, dt.deployment_id, COALESCE(w.k8s_namespace, '') AS k8s_namespace
FROM deployment_topology dt
LEFT JOIN deployments d ON d.id = dt.deployment_id
LEFT JOIN workspaces w ON w.id = dt.workspace_id
WHERE dt.region_id = sqlc.arg(region_id)
  AND dt.pk > sqlc.arg(after_pk)
  AND d.id IS NULL
ORDER BY dt.pk
LIMIT ?;
