-- Create "build_step_logs_v1" table
CREATE TABLE `default`.`build_step_logs_v1` (
  `time` Int64 CODEC(Delta(8), LZ4),
  `workspace_id` String,
  `project_id` String,
  `deployment_id` String,
  `message` String,
  `step_id` String
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `project_id`, `deployment_id`, `step_id`) ORDER BY (`workspace_id`, `project_id`, `deployment_id`, `step_id`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalMonth(3) SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
-- Create "build_steps_v1" table
CREATE TABLE `default`.`build_steps_v1` (
  `step_id` String,
  `started_at` Int64 CODEC(Delta(8), LZ4),
  `completed_at` Int64 CODEC(Delta(8), LZ4),
  `workspace_id` String,
  `project_id` String,
  `deployment_id` String,
  `name` String,
  `cache` Bool,
  `error` String,
  `has_logs` Bool
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `project_id`, `deployment_id`) ORDER BY (`workspace_id`, `project_id`, `deployment_id`) TTL toDateTime(fromUnixTimestamp64Milli(started_at)) + toIntervalMonth(3) SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
