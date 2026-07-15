-- name: UpdateQuota :exec
-- Overwrites every column of a workspace's quota row (team included, unlike
-- UpsertQuota whose ON DUPLICATE KEY UPDATE leaves it untouched). Callers pass a
-- full set of values, so it fits full resets like `unkey dev stripe reset`:
-- resetting only the core quotas would leave a paid tier's elevated rate-limit
-- and Deploy-resource allowances behind.
UPDATE quota
SET requests_per_month = sqlc.arg(requests_per_month),
    audit_logs_retention_days = sqlc.arg(audit_logs_retention_days),
    logs_retention_days = sqlc.arg(logs_retention_days),
    team = sqlc.arg(team),
    ratelimit_api_limit = sqlc.arg(ratelimit_api_limit),
    ratelimit_api_duration = sqlc.arg(ratelimit_api_duration),
    allocated_cpu_millicores_total = sqlc.arg(allocated_cpu_millicores_total),
    allocated_memory_mib_total = sqlc.arg(allocated_memory_mib_total),
    allocated_storage_mib_total = sqlc.arg(allocated_storage_mib_total),
    max_cpu_millicores_per_instance = sqlc.arg(max_cpu_millicores_per_instance),
    max_memory_mib_per_instance = sqlc.arg(max_memory_mib_per_instance),
    max_storage_mib_per_instance = sqlc.arg(max_storage_mib_per_instance),
    max_concurrent_builds = sqlc.arg(max_concurrent_builds)
WHERE workspace_id = sqlc.arg(workspace_id);
