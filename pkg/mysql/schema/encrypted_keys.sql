CREATE TABLE `encrypted_keys` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`key_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	`encrypted` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	CONSTRAINT `encrypted_keys_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_id_idx` UNIQUE(`key_id`)
);

