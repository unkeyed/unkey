CREATE TABLE `unkey_root_keys` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`hash` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(256),
	`prefix` varchar(16) NOT NULL,
	`start` varchar(256) NOT NULL,
	`end` varchar(4) NOT NULL,
	`enabled` boolean NOT NULL,
	`expires` datetime(3),
	`created_at` bigint NOT NULL,
	`deleted_at` bigint,
	CONSTRAINT `unkey_root_keys_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unkey_root_keys_id_unique` UNIQUE(`id`),
	CONSTRAINT `unkey_root_keys_hash_unique` UNIQUE(`hash`)
);

CREATE INDEX `workspace_id_idx` ON `unkey_root_keys` (`workspace_id`);
