CREATE TABLE `roles` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
	`name` varchar(512) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `roles_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`name`,`workspace_id`)
);

CREATE INDEX `workspace_id_idx` ON `roles` (`workspace_id`);

CREATE INDEX `roles_project_id_idx` ON `roles` (`project_id`);

