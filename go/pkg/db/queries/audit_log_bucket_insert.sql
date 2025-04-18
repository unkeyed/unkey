-- name: InsertAuditLogBucket :exec
INSERT INTO `audit_log_bucket` (
    id,
    workspace_id,
    name,
    retention_days,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(name),
    sqlc.arg(retention_days),
    sqlc.arg(created_at)
);
