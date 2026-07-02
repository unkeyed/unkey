-- name: ListWorkspacesWithDeployBudget :many
-- Lists every enabled workspace that has set a Deploy spend budget: the opt-in
-- set the spend-cap check evaluates. The check prices each one's month-to-date
-- Deploy usage and compares the net-of-credit overage against the budget.
-- org_id resolves the alert recipients (org admins via WorkOS); the included
-- credit is the per-period allowance subtracted from gross usage; the stop flag
-- decides whether 100% triggers teardown once enforcement (ENG-2923) lands.
SELECT
   w.id,
   w.name,
   w.slug,
   w.org_id,
   w.deploy_spend_budget_cents,
   w.deploy_spend_budget_stop,
   w.deploy_included_credit_cents
FROM `workspaces` w
WHERE w.deploy_spend_budget_cents IS NOT NULL
  AND w.enabled = true
  AND w.deleted_at_m IS NULL;
