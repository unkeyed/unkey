-- name: UpsertLimit :exec
INSERT INTO `limits` (
    workspace_id,
    api_billable_operations_count_max_per_month,
    api_requests_count_max_per_minute,
    logs_retention_days_max,
    logs_audit_retention_days_max,
    team_enabled,
    cpu_cores_max,
    cpu_cores_max_per_instance,
    memory_mib_max,
    memory_mib_max_per_instance,
    storage_mib_max,
    storage_mib_max_per_instance,
    builds_concurrent_max,
    custom_domains_max,
    autoscaling_replicas_max
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    api_billable_operations_count_max_per_month = VALUES(api_billable_operations_count_max_per_month),
    api_requests_count_max_per_minute = VALUES(api_requests_count_max_per_minute),
    logs_retention_days_max = VALUES(logs_retention_days_max),
    logs_audit_retention_days_max = VALUES(logs_audit_retention_days_max),
    team_enabled = VALUES(team_enabled),
    cpu_cores_max = VALUES(cpu_cores_max),
    cpu_cores_max_per_instance = VALUES(cpu_cores_max_per_instance),
    memory_mib_max = VALUES(memory_mib_max),
    memory_mib_max_per_instance = VALUES(memory_mib_max_per_instance),
    storage_mib_max = VALUES(storage_mib_max),
    storage_mib_max_per_instance = VALUES(storage_mib_max_per_instance),
    builds_concurrent_max = VALUES(builds_concurrent_max),
    custom_domains_max = VALUES(custom_domains_max),
    autoscaling_replicas_max = VALUES(autoscaling_replicas_max);
