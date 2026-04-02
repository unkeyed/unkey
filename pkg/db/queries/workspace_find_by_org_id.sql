-- name: FindWorkspaceByOrgID :one
SELECT * FROM `workspaces`
WHERE org_id = sqlc.arg(org_id)
AND enabled = true;
