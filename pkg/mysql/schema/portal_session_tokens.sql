CREATE TABLE `portal_session_tokens` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`portal_config_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`external_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`permissions` json NOT NULL,
	`preview` boolean NOT NULL DEFAULT false,
	`exchanged_at` bigint,
	`expires_at` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `portal_session_tokens_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `portal_session_tokens_id_unique` UNIQUE(`id`)
);

CREATE INDEX `idx_workspace` ON `portal_session_tokens` (`workspace_id`);

CREATE INDEX `idx_expires` ON `portal_session_tokens` (`expires_at`);

