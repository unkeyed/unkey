-- name: UpsertQuota :exec
INSERT INTO quota (
    workspace_id,
    requests_per_month,
    audit_logs_retention_days,
    logs_retention_days,
    team,
    ratelimit_api_limit,
    ratelimit_api_duration
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    requests_per_month = VALUES(requests_per_month),
    audit_logs_retention_days = VALUES(audit_logs_retention_days),
    logs_retention_days = VALUES(logs_retention_days),
    ratelimit_api_limit = VALUES(ratelimit_api_limit),
    ratelimit_api_duration = VALUES(ratelimit_api_duration);
