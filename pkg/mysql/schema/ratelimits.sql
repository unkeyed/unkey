CREATE TABLE `ratelimits` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`key_id` varchar(256),
	`identity_id` varchar(256),
	`limit` int NOT NULL,
	`duration` bigint NOT NULL,
	`auto_apply` boolean NOT NULL DEFAULT false,
	CONSTRAINT `ratelimits_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimits_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_key_idx` UNIQUE(`key_id`,`name`),
	CONSTRAINT `unique_name_per_identity_idx` UNIQUE(`identity_id`,`name`)
);

