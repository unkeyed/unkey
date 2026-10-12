-- name: DeleteManyProjectsByWorkspaceID :exec
-- DeleteManyProjectsByWorkspaceID hard deletes every project in a workspace.
-- Intended for tests that reset reusable fixtures.
DELETE FROM projects
WHERE workspace_id = sqlc.arg(workspace_id);
