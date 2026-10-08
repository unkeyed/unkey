-- name: SetWorkspaceDeploySpendBudget :exec
-- SetWorkspaceDeploySpendBudget sets or clears a workspace's Deploy spend budget
-- and whether reaching it stops compute. Only tests use this query.
UPDATE `workspace_billing`
SET spend_budget_cents = sqlc.arg(spend_budget_cents),
    spend_budget_stop = sqlc.arg(spend_budget_stop)
WHERE workspace_id = sqlc.arg(workspace_id);
