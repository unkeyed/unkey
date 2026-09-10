-- name: SetWorkspaceDeployPlan :exec
UPDATE `workspace_billing`
SET plan = sqlc.arg(plan)
WHERE workspace_id = sqlc.arg(workspace_id);
