CREATE TABLE `ratelimits` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`key_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`identity_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`limit` bigint unsigned NOT NULL,
	`duration` bigint unsigned NOT NULL,
	`auto_apply` boolean NOT NULL DEFAULT false,
	CONSTRAINT `ratelimits_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimits_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_key_idx` UNIQUE(`key_id`,`name`),
	CONSTRAINT `unique_name_per_identity_idx` UNIQUE(`identity_id`,`name`)
);

