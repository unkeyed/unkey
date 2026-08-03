-- name: FindQuotaByWorkspaceID :one
SELECT quota.pk, quota.workspace_id, quota.requests_per_month, quota.logs_retention_days, quota.audit_logs_retention_days, quota.team, quota.ratelimit_api_limit, quota.ratelimit_api_duration, quota.allocated_cpu_millicores_total, quota.allocated_memory_mib_total, quota.allocated_storage_mib_total, quota.max_cpu_millicores_per_instance, quota.max_memory_mib_per_instance, quota.max_storage_mib_per_instance, quota.max_concurrent_builds, quota.max_replicas_per_region
FROM `quota`
WHERE workspace_id = sqlc.arg('workspace_id');
