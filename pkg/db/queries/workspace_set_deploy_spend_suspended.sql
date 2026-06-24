-- name: SetWorkspaceDeploySpendSuspended :exec
-- Records whether the spend cap has suspended a workspace's compute. Written by
-- the spend-cap check on the suspend/resume transition; read by the orchestrator
-- (to keep checking a suspended workspace even after its budget is removed) and
-- the dashboard (to show a suspended state).
UPDATE `workspaces`
SET deploy_spend_suspended = sqlc.arg(suspended),
    updated_at_m = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
