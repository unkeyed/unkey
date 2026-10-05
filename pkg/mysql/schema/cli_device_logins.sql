CREATE TABLE `cli_device_logins` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`user_code` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`device_code` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workos_verification_uri` varchar(2048) NOT NULL,
	`poll_interval_seconds` int NOT NULL,
	`expires_at` bigint NOT NULL,
	`status` varchar(32) NOT NULL,
	`device_name` varchar(512),
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`approver_user_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`approver_name` varchar(256),
	`approver_roles` json,
	`permissions` json,
	`key_name` varchar(256),
	`created_at` bigint NOT NULL,
	`approved_at` bigint,
	CONSTRAINT `cli_device_logins_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `cli_device_logins_id_unique` UNIQUE(`id`),
	CONSTRAINT `cli_device_logins_user_code_unique` UNIQUE(`user_code`)
);

CREATE INDEX `cli_device_logins_expires_at_idx` ON `cli_device_logins` (`expires_at`);
