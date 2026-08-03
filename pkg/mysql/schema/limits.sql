CREATE TABLE `limits` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`api_billable_operations_count_max_per_month` bigint unsigned NOT NULL,
	`api_requests_count_max_per_minute` bigint unsigned,
	`logs_retention_days_max` bigint unsigned NOT NULL,
	`logs_audit_retention_days_max` bigint unsigned NOT NULL,
	`team_enabled` boolean NOT NULL,
	`cpu_max` bigint unsigned NOT NULL,
	`cpu_max_per_instance` bigint unsigned NOT NULL,
	`memory_mib_max` bigint unsigned NOT NULL,
	`memory_mib_max_per_instance` bigint unsigned NOT NULL,
	`disk_ephemeral_mib_max` bigint unsigned NOT NULL,
	`disk_ephemeral_mib_max_per_instance` bigint unsigned NOT NULL,
	`builds_concurrent_count_max` bigint unsigned NOT NULL,
	`custom_domains_count_max` bigint unsigned NOT NULL,
	CONSTRAINT `limits_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `limits_workspace_id_unique` UNIQUE(`workspace_id`)
);

