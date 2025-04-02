-- name: FindAuditLogBucketIDByWorkspaceIDAndName :one
SELECT id FROM audit_log_bucket WHERE workspace_id = sqlc.arg(workspace_id) AND name = sqlc.arg(name);
