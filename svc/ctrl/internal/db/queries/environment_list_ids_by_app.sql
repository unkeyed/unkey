-- name: ListEnvironmentIdsByApp :many
SELECT environments.id FROM environments WHERE environments.app_id = sqlc.arg(app_id)
UNION
SELECT deployments.environment_id AS id FROM deployments WHERE deployments.app_id = sqlc.arg(app_id);
