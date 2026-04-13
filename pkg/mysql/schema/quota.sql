CREATE TABLE `quota` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`requests_per_month` bigint NOT NULL DEFAULT 0,
	`logs_retention_days` int NOT NULL DEFAULT 0,
	`audit_logs_retention_days` int NOT NULL DEFAULT 0,
	`team` boolean NOT NULL DEFAULT false,
	`ratelimit_api_limit` int unsigned,
	`ratelimit_api_duration` int unsigned,
	`allocated_cpu_millicores_total` int unsigned NOT NULL DEFAULT 10000,
	`allocated_memory_mib_total` int unsigned NOT NULL DEFAULT 20480,
	`allocated_storage_mib_total` int unsigned NOT NULL DEFAULT 51200,
	`max_cpu_millicores_per_instance` int unsigned NOT NULL DEFAULT 2000,
	`max_memory_mib_per_instance` int unsigned NOT NULL DEFAULT 4096,
	`max_storage_mib_per_instance` int unsigned NOT NULL DEFAULT 10240,
	CONSTRAINT `quota_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `quota_workspace_id_unique` UNIQUE(`workspace_id`)
);

