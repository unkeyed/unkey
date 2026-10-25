CREATE TABLE `notifications` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`source_event_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`notification_type` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`request_hash` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`schema_version` int unsigned NOT NULL,
	`resource_context` json NOT NULL,
	`payload` json NOT NULL,
	`occurred_at` bigint NOT NULL,
	`expires_at` bigint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notifications_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notifications_id_unique` UNIQUE(`id`),
	CONSTRAINT `notifications_source_event_idx` UNIQUE(`workspace_id`,`notification_type`,`source_event_id`)
);

CREATE INDEX `notifications_workspace_created_idx` ON `notifications` (`workspace_id`,`created_at`,`id`);

