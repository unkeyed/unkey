CREATE TABLE `acme_users` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`encrypted_key` text NOT NULL,
	`registration_uri` text,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_users_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `acme_users_id_unique` UNIQUE(`id`)
);

CREATE INDEX `domain_idx` ON `acme_users` (`workspace_id`);

