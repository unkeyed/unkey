-- name: ListAppIdsByProject :many
SELECT apps.id FROM apps WHERE apps.project_id = sqlc.arg(project_id)
UNION
SELECT environments.app_id AS id FROM environments WHERE environments.project_id = sqlc.arg(project_id)
UNION
SELECT deployments.app_id AS id FROM deployments WHERE deployments.project_id = sqlc.arg(project_id);
