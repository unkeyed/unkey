CREATE TABLE `key_migrations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(36) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(36) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`algorithm` enum('sha256','github.com/seamapi/prefixed-api-key') NOT NULL,
	CONSTRAINT `key_migrations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_migrations_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_id_per_workspace_id` UNIQUE(`id`,`workspace_id`)
);

