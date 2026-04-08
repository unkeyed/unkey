CREATE TABLE `projects` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`depot_project_id` varchar(255),
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `projects_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `projects_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_slug_idx` UNIQUE(`workspace_id`,`slug`)
);

