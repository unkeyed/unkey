CREATE TABLE `portals` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`slug` varchar(64) NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`key_auth_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`enabled` boolean NOT NULL DEFAULT true,
	`logo_url` varchar(500),
	`primary_color` varchar(7),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `portals_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `portals_id_unique` UNIQUE(`id`),
	CONSTRAINT `idx_workspace_slug` UNIQUE(`workspace_id`,`slug`),
	CONSTRAINT `idx_app_id` UNIQUE(`app_id`),
	CONSTRAINT `idx_key_auth_id` UNIQUE(`key_auth_id`)
);

