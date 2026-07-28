CREATE TABLE `keys_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`key_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`permission_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_permissions_key_id_permission_id_workspace_id` UNIQUE(`key_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `key_id_permission_id_idx` UNIQUE(`key_id`,`permission_id`)
);

