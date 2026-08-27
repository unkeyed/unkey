CREATE TABLE `logdrain_state` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`logdrain_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`status` enum('active','paused_by_failure') NOT NULL DEFAULT 'active',
	`consecutive_failures` int NOT NULL DEFAULT 0,
	`committed_offset_inserted_at` bigint NOT NULL DEFAULT 0,
	`committed_offset_event_id` varchar(255) COLLATE utf8mb4_0900_bin NOT NULL DEFAULT '',
	`next_attempt_at` bigint NOT NULL DEFAULT 0,
	`last_error` text,
	`lease_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
	`fencing_token` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
	`lease_expires_at` bigint NOT NULL DEFAULT 0,
	CONSTRAINT `logdrain_state_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `logdrain_state_logdrain_id_unique` UNIQUE(`logdrain_id`)
);

CREATE INDEX `lease_expires_at_logdrain_id_idx` ON `logdrain_state` (`lease_expires_at`,`logdrain_id`);

CREATE INDEX `lease_id_status_next_attempt_at_logdrain_id_idx` ON `logdrain_state` (`lease_id`,`status`,`next_attempt_at`,`logdrain_id`);
