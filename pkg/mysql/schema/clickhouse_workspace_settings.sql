CREATE TABLE `clickhouse_workspace_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`username` varchar(256) NOT NULL,
	`password_encrypted` text NOT NULL,
	`quota_duration_seconds` int NOT NULL DEFAULT 3600,
	`max_queries_per_window` int NOT NULL DEFAULT 1000,
	`max_execution_time_per_window` int NOT NULL DEFAULT 1800,
	`max_query_execution_time` int NOT NULL DEFAULT 30,
	`max_query_memory_bytes` bigint NOT NULL DEFAULT 1000000000,
	`max_query_result_rows` int NOT NULL DEFAULT 10000,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	CONSTRAINT `clickhouse_workspace_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `clickhouse_workspace_settings_workspace_id_unique` UNIQUE(`workspace_id`),
	CONSTRAINT `clickhouse_workspace_settings_username_unique` UNIQUE(`username`)
);

