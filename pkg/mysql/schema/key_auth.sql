CREATE TABLE `key_auth` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`store_encrypted_keys` boolean NOT NULL DEFAULT false,
	`default_prefix` varchar(8),
	`default_bytes` int DEFAULT 16,
	`size_approx` int NOT NULL DEFAULT 0,
	`size_last_updated_at` bigint NOT NULL DEFAULT 0,
	CONSTRAINT `key_auth_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_auth_id_unique` UNIQUE(`id`)
);

