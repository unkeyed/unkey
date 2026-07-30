-- name: ListWorkspacesWithDeployBudget :many
-- Lists every enabled workspace that has set a Deploy spend budget, plus any
-- that is currently spend-cap suspended even without a budget: the set the
-- spend-cap check evaluates. The check prices each one's month-to-date Deploy
-- usage and compares the gross total spend against the budget. Suspended
-- workspaces are included even without a budget so the check can resume them
-- after the budget is removed (otherwise removing the budget would drop them
-- from this list and they would never resume).
-- org_id resolves the alert recipients (org admins via WorkOS); the stop flag
-- decides whether 100% triggers teardown; deploy_spend_suspended tells the check
-- whether the cap has already stopped this workspace's compute.
SELECT
   w.id,
   w.name,
   w.slug,
   w.org_id,
   b.spend_budget_cents,
   b.spend_budget_stop,
   b.spend_suspended
FROM `workspaces` w
LEFT JOIN `workspace_billing` b ON w.id = b.workspace_id
WHERE (b.spend_budget_cents IS NOT NULL OR b.spend_suspended = TRUE)
  AND w.enabled = true
  AND w.deleted_at_m IS NULL;
