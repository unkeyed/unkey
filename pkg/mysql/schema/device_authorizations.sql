CREATE TABLE `device_authorizations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`user_code` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`expires_at` bigint NOT NULL,
	`status` enum('pending','approved','consumed','denied','expired') NOT NULL,
	`device_name` varchar(512),
	`requester_ip` varchar(64),
	`requester_user_agent` varchar(512),
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`approver_user_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`approver_name` varchar(256),
	`approver_roles` json,
	`permissions` json,
	`session_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`created_at` bigint NOT NULL,
	`approved_at` bigint,
	`client_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`redirect_uri` varchar(2048) COLLATE utf8mb4_0900_as_cs,
	`redirect_uri_presented` boolean,
	`state` varchar(1024) COLLATE utf8mb4_0900_as_cs,
	`code_challenge` varchar(128) COLLATE utf8mb4_0900_as_cs,
	`code_hash` varchar(64) COLLATE utf8mb4_0900_as_cs,
	CONSTRAINT `device_authorizations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `device_authorizations_id_unique` UNIQUE(`id`),
	CONSTRAINT `device_authorizations_user_code_unique` UNIQUE(`user_code`),
	CONSTRAINT `device_authorizations_session_id_unique` UNIQUE(`session_id`),
	CONSTRAINT `device_authorizations_code_hash_unique` UNIQUE(`code_hash`)
);

CREATE INDEX `device_authorizations_expires_at_idx` ON `device_authorizations` (`expires_at`);

