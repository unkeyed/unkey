-- name: ListDepotProjectIDsByWorkspaceID :many
-- Lists the Depot projects a workspace's projects build in, soft-deleted
-- projects included, so a GDPR erasure can delete them.
SELECT depot_project_id FROM projects
WHERE workspace_id = sqlc.arg(workspace_id)
  AND depot_project_id IS NOT NULL
  AND depot_project_id != ''
ORDER BY id;
