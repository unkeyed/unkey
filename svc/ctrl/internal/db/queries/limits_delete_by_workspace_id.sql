-- name: DeleteLimitsByWorkspaceId :exec
-- DeleteLimitsByWorkspaceId removes a workspace's limits row, so a test can
-- exercise the state before billing has written it. Only tests use this query.
DELETE FROM `limits`
WHERE workspace_id = sqlc.arg(workspace_id);
