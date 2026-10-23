-- name: FindDeploymentWithApp :one
SELECT d.id, d.app_id, a.current_deployment_id
FROM deployments d
JOIN apps a ON a.id = d.app_id
JOIN environments e ON e.id = d.environment_id
JOIN projects p ON p.id = d.project_id
WHERE d.id = sqlc.arg(id)
  AND e.deleted_at IS NULL
  AND a.deleted_at IS NULL
  AND p.deleted_at IS NULL
LOCK IN SHARE MODE;
