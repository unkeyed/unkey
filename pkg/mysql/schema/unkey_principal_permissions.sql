CREATE TABLE `unkey_principal_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`principal_type` enum('root_key','oidc') NOT NULL,
	`principal_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`slug` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `unkey_principal_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unkey_principal_permissions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unkey_principal_permissions_principal_slug_idx` UNIQUE(`workspace_id`,`principal_type`,`principal_id`,`slug`)
);

