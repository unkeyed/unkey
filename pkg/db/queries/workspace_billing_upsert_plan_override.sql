-- name: UpsertWorkspaceBillingPlanOverride :exec
INSERT INTO `workspace_billing` (
    workspace_id,
    plan_override,
    created_at_m
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.narg(plan_override),
    sqlc.arg(created_at_m)
)
ON DUPLICATE KEY UPDATE
    plan_override = VALUES(plan_override);
