CREATE TABLE `user_device_sessions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`user_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`user_name` varchar(256),
	`permissions` json NOT NULL,
	`device_name` varchar(512),
	`refresh_token_hash` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`previous_refresh_token_hash` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`refresh_family` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`rotated_at` bigint,
	`created_at` bigint NOT NULL,
	`last_used_at` bigint NOT NULL,
	`expires_at` bigint NOT NULL,
	`revoked_at` bigint,
	`oauth_client_id` varchar(512) COLLATE utf8mb4_0900_as_cs,
	`audience` varchar(512) COLLATE utf8mb4_0900_as_cs,
	CONSTRAINT `user_device_sessions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `user_device_sessions_id_unique` UNIQUE(`id`),
	CONSTRAINT `user_device_sessions_refresh_token_hash_unique` UNIQUE(`refresh_token_hash`),
	CONSTRAINT `user_device_sessions_previous_refresh_token_hash_unique` UNIQUE(`previous_refresh_token_hash`)
);

CREATE INDEX `user_device_sessions_workspace_user_created_idx` ON `user_device_sessions` (`workspace_id`,`user_id`,`created_at`);

CREATE INDEX `user_device_sessions_refresh_family_idx` ON `user_device_sessions` (`refresh_family`);

CREATE INDEX `user_device_sessions_expires_at_idx` ON `user_device_sessions` (`expires_at`);

