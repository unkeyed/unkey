-- name: InsertAuditLogTarget :exec
INSERT INTO `audit_log_target` (
    workspace_id,
    bucket_id,
    audit_log_id,
    display_name,
    type,
    id,
    name,
    meta,
    created_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(bucket_id),
    sqlc.arg(audit_log_id),
    sqlc.arg(display_name),
    sqlc.arg(type),
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(meta),
    sqlc.arg(created_at)
);
