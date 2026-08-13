CREATE TABLE `portal_sessions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`portal_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`external_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`scopes` json NOT NULL,
	`preview` boolean NOT NULL DEFAULT false,
	`exchange_code_hash` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`exchange_code_expires_at` bigint NOT NULL,
	`access_token_hash` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`access_token_created_at` bigint,
	`access_token_expires_at` bigint,
	`revoked_at` bigint,
	`created_at` bigint NOT NULL,
	CONSTRAINT `portal_sessions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `portal_sessions_id_unique` UNIQUE(`id`),
	CONSTRAINT `idx_exchange_code_hash` UNIQUE(`exchange_code_hash`),
	CONSTRAINT `idx_access_token_hash` UNIQUE(`access_token_hash`)
);

CREATE INDEX `idx_workspace` ON `portal_sessions` (`workspace_id`);

CREATE INDEX `idx_external_id` ON `portal_sessions` (`external_id`);

CREATE INDEX `idx_exchange_code_expires` ON `portal_sessions` (`exchange_code_expires_at`);

CREATE INDEX `idx_access_token_expires` ON `portal_sessions` (`access_token_expires_at`);

