-- name: FindAuditLogTargetByID :many
SELECT sqlc.embed(audit_log_target), sqlc.embed(audit_log)
FROM audit_log_target
JOIN audit_log ON audit_log.id = audit_log_target.audit_log_id
WHERE audit_log_target.id = sqlc.arg(id);
