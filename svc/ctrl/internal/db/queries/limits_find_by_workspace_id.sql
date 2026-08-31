-- name: FindLimitsByWorkspaceID :one
SELECT pk, workspace_id, api_billable_operations_count_max_per_month, api_requests_count_max_per_minute, logs_retention_days_max, logs_audit_retention_days_max, team_enabled, cpu_cores_max, cpu_cores_max_per_instance, memory_mib_max, memory_mib_max_per_instance, storage_mib_max, storage_mib_max_per_instance, builds_concurrent_max, custom_domains_max, autoscaling_replicas_max
FROM `limits`
WHERE workspace_id = sqlc.arg('workspace_id');
