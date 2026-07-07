-- name: ListDeployments :many
-- has_status_filter gates the status clause; without it sqlc renders an empty
-- status set as IN (NULL), which matches nothing.
SELECT d.* FROM `deployments` d
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR d.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR d.app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR d.environment_id = sqlc.arg(environment_id))
  AND (sqlc.arg(has_status_filter) = FALSE OR d.status IN (sqlc.slice('statuses')))
  AND (
    sqlc.arg(cursor_id) = ''
    OR d.pk < (SELECT c.pk FROM `deployments` c WHERE c.id = sqlc.arg(cursor_id))
  )
ORDER BY d.pk DESC
LIMIT ?;
