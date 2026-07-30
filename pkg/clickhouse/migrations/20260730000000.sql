-- Add a stable identity to runtime logs.
--
-- Vector must be rolled out first. Its ClickHouse sink skips unknown fields,
-- so log_id is safely ignored before this migration. After the migration,
-- Vector's Unkey-style ID is retained across sink retries and gives every new
-- log a stable identity. Writers without log_id insert an empty string.
--
-- Existing rows also read log_id as an empty string. Runtime-log readers retain
-- their content-based identity fallback until those rows expire.
--
-- Appending a newly added column to ORDER BY is a metadata-only ALTER. As
-- required by ClickHouse, log_id is added without a DEFAULT and in the same
-- ALTER that modifies the sorting key. The existing primary key remains a
-- prefix of the new sorting key.

ALTER TABLE `default`.`runtime_logs_raw_v1`
  ADD COLUMN `log_id` String CODEC(ZSTD(1)) AFTER `time`,
  MODIFY ORDER BY (`workspace_id`, `project_id`, `environment_id`, `app_id`, `time`, `deployment_id`, `log_id`);
