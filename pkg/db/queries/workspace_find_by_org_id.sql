-- name: FindWorkspaceByOrgID :one
SELECT * FROM `workspaces`
WHERE org_id = sqlc.arg(org_id)
AND deleted_at_m IS NULL;
