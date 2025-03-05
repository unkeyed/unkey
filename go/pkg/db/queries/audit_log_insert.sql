-- name: InsertAuditLog :exec
INSERT INTO `audit_log` (
    id,
    workspace_id,
    bucket_id,
    event,
    time,
    display,
    remote_ip,
    user_agent,
    actor_type,
    actor_id,
    actor_name,
    actor_meta,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(bucket_id),
    sqlc.arg(event),
    sqlc.arg(time),
    sqlc.arg(display),
    sqlc.arg(remote_ip),
    sqlc.arg(user_agent),
    sqlc.arg(actor_type),
    sqlc.arg(actor_id),
    sqlc.arg(actor_name),
    sqlc.arg(actor_meta),
    sqlc.arg(created_at)
);
