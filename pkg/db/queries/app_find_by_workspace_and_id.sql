-- name: FindAppByWorkspaceAndId :one
SELECT *
FROM apps
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);
