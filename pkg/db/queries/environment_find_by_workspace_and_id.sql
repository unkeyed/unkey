-- name: FindEnvironmentByWorkspaceAndId :one
SELECT *
FROM environments
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);
