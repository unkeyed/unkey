CREATE TABLE `oidc_vercel_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`oidc_vercel_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`permission_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `oidc_vercel_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `oidc_vercel_permission_idx` UNIQUE(`oidc_vercel_id`,`permission_id`)
);

CREATE INDEX `permission_id_idx` ON `oidc_vercel_permissions` (`permission_id`);
