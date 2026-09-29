CREATE TABLE `unkey_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`for_workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`principal_type` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`principal_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(512) NOT NULL,
	`slug` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `unkey_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unkey_permissions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unkey_permissions_principal_slug_idx` UNIQUE(`for_workspace_id`,`principal_type`,`principal_id`,`slug`)
);

