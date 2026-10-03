-- name: ConfirmDeploymentTopologyRemoval :execrows
DELETE dt FROM deployment_topology dt
LEFT JOIN deployments d ON d.id = dt.deployment_id
LEFT JOIN projects p ON p.id = d.project_id
LEFT JOIN apps a ON a.id = d.app_id
LEFT JOIN environments e ON e.id = d.environment_id
WHERE dt.deployment_id = sqlc.arg(deployment_id)
  AND dt.region_id = sqlc.arg(region_id)
  AND (d.id IS NULL OR p.id IS NULL OR a.id IS NULL OR e.id IS NULL
    OR p.deleting_at IS NOT NULL OR a.deleting_at IS NOT NULL OR e.deleting_at IS NOT NULL);
