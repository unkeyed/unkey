CREATE TABLE `logdrains` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(128) NOT NULL,
	`stream` enum('audit_logs') NOT NULL,
	`config` longblob NOT NULL,
	`enabled` boolean NOT NULL DEFAULT true,
	`status` enum('active','paused_by_failure') NOT NULL DEFAULT 'active',
	`consecutive_failures` int NOT NULL DEFAULT 0,
	`committed_offset_inserted_at` bigint NOT NULL DEFAULT 0,
	`committed_offset_event_id` varchar(255) COLLATE utf8mb4_0900_bin NOT NULL DEFAULT '',
	`next_attempt_at` bigint NOT NULL DEFAULT 0,
	`lease_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`fencing_token` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`lease_expires_at` bigint NOT NULL DEFAULT 0,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `logdrains_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `logdrains_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_id_idx` ON `logdrains` (`workspace_id`);

CREATE INDEX `lease_expires_at_id_idx` ON `logdrains` (`lease_expires_at`,`id`);

CREATE INDEX `lease_id_status_next_attempt_at_id_idx` ON `logdrains` (`lease_id`,`status`,`next_attempt_at`,`id`);
