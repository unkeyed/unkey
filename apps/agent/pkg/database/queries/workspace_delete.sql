-- name: DeleteWorkspace :exec
DELETE FROM `workspaces` WHERE id = sqlc.arg("workspace_id");
