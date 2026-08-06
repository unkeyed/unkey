CREATE TABLE `ratelimit_global_counters` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`namespace` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`identifier` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`duration_ms` bigint unsigned NOT NULL,
	`sequence` bigint NOT NULL,
	`region` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`count` bigint unsigned NOT NULL,
	`expires_at` bigint unsigned NOT NULL,
	`updated_at` bigint unsigned NOT NULL,
	CONSTRAINT `ratelimit_global_counters_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_window_region` UNIQUE(`workspace_id`,`namespace`,`identifier`,`duration_ms`,`sequence`,`region`)
);

CREATE INDEX `expires_at_idx` ON `ratelimit_global_counters` (`expires_at`);

