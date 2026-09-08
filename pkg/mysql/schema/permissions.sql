CREATE TABLE `permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(512) NOT NULL,
	`slug` varchar(512) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `permissions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_slug_per_workspace_idx` UNIQUE(`workspace_id`,`slug`),
	CONSTRAINT `unique_slug_per_project_idx` UNIQUE(`project_id`,`slug`)
);

CREATE INDEX `permissions_workspace_id_idx` ON `permissions` (`workspace_id`);

CREATE INDEX `permissions_project_id_idx` ON `permissions` (`project_id`);
