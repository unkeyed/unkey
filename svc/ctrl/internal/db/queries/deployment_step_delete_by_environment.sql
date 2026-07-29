-- name: DeleteDeploymentStepsByEnvironmentId :exec
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
DELETE ds FROM deployment_steps ds
JOIN deployments d ON (d.id COLLATE utf8mb4_0900_ai_ci = ds.deployment_id AND d.id COLLATE utf8mb4_0900_as_cs = ds.deployment_id)
WHERE d.environment_id = sqlc.arg(environment_id);
