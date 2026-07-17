-- name: ClearWorkspaceDeployPlan :exec
-- Clears the local Deploy entitlement mirror on cancel. Leaves the Stripe
-- linkage (customer/subscription) intact: a mixed subscription keeps running for
-- the API plan, and a Deploy-only subscription cancels at period end. After this
-- the invoice.created webhook and the month-end close both skip the workspace
-- (they require deploy_plan IS NOT NULL), so the final invoice auto-finalizes.
UPDATE `workspaces`
SET deploy_plan = NULL,
    updated_at_m = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
