-- name: InsertDeploymentApproval :exec
INSERT INTO `deployment_approvals` (
    deployment_id,
    approved_by,
    approved_at,
    sender_login
)
VALUES (
    sqlc.arg(deployment_id),
    sqlc.arg(approved_by),
    sqlc.arg(approved_at),
    sqlc.arg(sender_login)
);
