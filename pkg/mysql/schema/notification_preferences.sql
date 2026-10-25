CREATE TABLE `notification_preferences` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`user_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`notification_type` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`channel` enum('email') NOT NULL,
	`enabled` boolean NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notification_preferences_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notification_preferences_id_unique` UNIQUE(`id`),
	CONSTRAINT `notification_preferences_user_type_channel_idx` UNIQUE(`workspace_id`,`user_id`,`notification_type`,`channel`)
);

