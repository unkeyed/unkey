-- name: UpsertQuota :exec
INSERT INTO quota (
    workspace_id,
    requests_per_month,
    audit_logs_retention_days,
    logs_retention_days,
    team
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    requests_per_month = VALUES(requests_per_month),
    audit_logs_retention_days = VALUES(audit_logs_retention_days),
    logs_retention_days = VALUES(logs_retention_days);
