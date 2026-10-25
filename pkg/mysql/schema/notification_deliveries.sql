CREATE TABLE `notification_deliveries` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`notification_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`channel` enum('email','discord','slack','webhook') NOT NULL,
	`target_key` varchar(320) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`encrypted_target_snapshot` mediumtext NOT NULL,
	`provider` varchar(64) COLLATE utf8mb4_0900_as_cs,
	`provider_message_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`status` enum('pending','sending','accepted','delivered','failed','suppressed','unknown') NOT NULL DEFAULT 'pending',
	`reason` varchar(256),
	`attempt_count` int unsigned NOT NULL DEFAULT 0,
	`first_attempt_at` bigint,
	`next_attempt_at` bigint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notification_deliveries_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notification_deliveries_id_unique` UNIQUE(`id`),
	CONSTRAINT `notification_deliveries_target_idx` UNIQUE(`notification_id`,`channel`,`target_key`)
);

CREATE INDEX `notification_deliveries_workspace_notification_idx` ON `notification_deliveries` (`workspace_id`,`notification_id`);

CREATE INDEX `notification_deliveries_provider_message_idx` ON `notification_deliveries` (`provider`,`provider_message_id`);

