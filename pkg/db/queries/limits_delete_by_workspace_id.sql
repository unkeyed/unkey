-- name: DeleteLimitByWorkspaceID :exec
-- DeleteLimitByWorkspaceID removes a workspace's limits row. Intended for
-- tests of workspaces without configured limits.
DELETE FROM `limits`
WHERE workspace_id = sqlc.arg(workspace_id);
