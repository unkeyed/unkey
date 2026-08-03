-- Backfill limits from quota after the table and dual-write code are live.
-- quota remains the read source during this phase. Run the checks before and
-- after the insert and investigate every returned row. These SELECT statements
-- report problems; they do not abort this script or the INSERT.

-- PRE: must return no rows. A rate limit is either absent (both NULL) or exactly
-- a one-minute pair. The backfill deliberately excludes invalid rows.
SELECT `workspace_id`, `ratelimit_api_limit`, `ratelimit_api_duration`
FROM `quota`
WHERE (`ratelimit_api_limit` IS NULL) <> (`ratelimit_api_duration` IS NULL)
   OR (`ratelimit_api_duration` IS NOT NULL AND `ratelimit_api_duration` <> 60000);

-- PRE: must return no rows. The target is unsigned and the dashboard reads
-- bigint values as JavaScript numbers while dual-write is active.
SELECT
  `workspace_id`,
  `requests_per_month`,
  `logs_retention_days`,
  `audit_logs_retention_days`
FROM `quota`
WHERE `requests_per_month` < 0
   OR `requests_per_month` > 9007199254740991
   OR `logs_retention_days` < 0
   OR `audit_logs_retention_days` < 0;

-- PRE: must return no rows. Ceil division converts legacy millicores to whole
-- vCPU without reducing custom entitlements and must remain a safe integer.
SELECT `workspace_id`, `allocated_cpu_millicores_total`, `max_cpu_millicores_per_instance`
FROM `quota`
WHERE `allocated_cpu_millicores_total` < 0
   OR `max_cpu_millicores_per_instance` < 0
   OR CEIL(`allocated_cpu_millicores_total` / 1000) > 9007199254740991
   OR CEIL(`max_cpu_millicores_per_instance` / 1000) > 9007199254740991;

-- Insert only missing rows. Rows already written by the live dual-writer may be
-- newer than this statement's source snapshot and must not be overwritten.
INSERT IGNORE INTO `limits` (
  `workspace_id`,
  `api_billable_operations_count_max_per_month`,
  `api_requests_count_max_per_minute`,
  `logs_retention_days_max`,
  `logs_audit_retention_days_max`,
  `team_enabled`,
  `cpu_cores_max`,
  `cpu_cores_max_per_instance`,
  `memory_mib_max`,
  `memory_mib_max_per_instance`,
  `disk_ephemeral_mib_max`,
  `disk_ephemeral_mib_max_per_instance`,
  `builds_concurrent_count_max`,
  `custom_domains_count_max`
)
SELECT
  q.`workspace_id`,
  q.`requests_per_month`,
  q.`ratelimit_api_limit`,
  q.`logs_retention_days`,
  q.`audit_logs_retention_days`,
  q.`team`,
  CEIL(q.`allocated_cpu_millicores_total` / 1000),
  CEIL(q.`max_cpu_millicores_per_instance` / 1000),
  q.`allocated_memory_mib_total`,
  q.`max_memory_mib_per_instance`,
  q.`allocated_storage_mib_total`,
  q.`max_storage_mib_per_instance`,
  q.`max_concurrent_builds`,
  CASE COALESCE(NULLIF(b.`plan_override`, ''), NULLIF(b.`plan`, ''))
    WHEN 'starter' THEN 1
    WHEN 'pro' THEN 1000000
    WHEN 'business' THEN 1000000
    ELSE 0
  END
FROM `quota` q
LEFT JOIN `workspace_billing` b ON b.`workspace_id` = q.`workspace_id`
WHERE (
    (q.`ratelimit_api_limit` IS NULL AND q.`ratelimit_api_duration` IS NULL)
    OR (q.`ratelimit_api_limit` IS NOT NULL AND q.`ratelimit_api_duration` = 60000)
  )
  AND q.`requests_per_month` BETWEEN 0 AND 9007199254740991
  AND q.`logs_retention_days` >= 0
  AND q.`audit_logs_retention_days` >= 0;

-- POST: every workspace must have one quota row and one limits row.
SELECT w.`id`
FROM `workspaces` w
LEFT JOIN `quota` q ON q.`workspace_id` = w.`id`
LEFT JOIN `limits` l ON l.`workspace_id` = w.`id`
WHERE q.`workspace_id` IS NULL OR l.`workspace_id` IS NULL;

-- POST: limits must not contain a row without a legacy dual-write source.
SELECT l.`workspace_id`
FROM `limits` l
LEFT JOIN `quota` q ON q.`workspace_id` = l.`workspace_id`
WHERE q.`workspace_id` IS NULL;

-- POST: must return no rows. This verifies every mapped value.
SELECT q.`workspace_id`
FROM `quota` q
JOIN `limits` l ON l.`workspace_id` = q.`workspace_id`
LEFT JOIN `workspace_billing` b ON b.`workspace_id` = q.`workspace_id`
WHERE NOT (l.`api_billable_operations_count_max_per_month` <=> q.`requests_per_month`)
   OR NOT (l.`api_requests_count_max_per_minute` <=> q.`ratelimit_api_limit`)
   OR NOT (l.`logs_retention_days_max` <=> q.`logs_retention_days`)
   OR NOT (l.`logs_audit_retention_days_max` <=> q.`audit_logs_retention_days`)
   OR NOT (l.`team_enabled` <=> q.`team`)
   OR NOT (l.`cpu_cores_max` <=> CEIL(q.`allocated_cpu_millicores_total` / 1000))
   OR NOT (l.`cpu_cores_max_per_instance` <=> CEIL(q.`max_cpu_millicores_per_instance` / 1000))
   OR NOT (l.`memory_mib_max` <=> q.`allocated_memory_mib_total`)
   OR NOT (l.`memory_mib_max_per_instance` <=> q.`max_memory_mib_per_instance`)
   OR NOT (l.`disk_ephemeral_mib_max` <=> q.`allocated_storage_mib_total`)
   OR NOT (l.`disk_ephemeral_mib_max_per_instance` <=> q.`max_storage_mib_per_instance`)
   OR NOT (l.`builds_concurrent_count_max` <=> q.`max_concurrent_builds`)
   OR NOT (
     l.`custom_domains_count_max` <=>
     CASE COALESCE(NULLIF(b.`plan_override`, ''), NULLIF(b.`plan`, ''))
       WHEN 'starter' THEN 1
       WHEN 'pro' THEN 1000000
       WHEN 'business' THEN 1000000
       ELSE 0
     END
   );

-- POST: must return no rows. limits must not contain orphan workspaces.
SELECT l.`workspace_id`
FROM `limits` l
LEFT JOIN `workspaces` w ON w.`id` = l.`workspace_id`
WHERE w.`id` IS NULL;
