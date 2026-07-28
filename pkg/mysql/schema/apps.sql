CREATE TABLE `apps` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(64) NOT NULL,
	`name` varchar(256) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`default_branch` varchar(256) NOT NULL DEFAULT 'main',
	`current_deployment_id` varchar(256),
	`is_rolled_back` boolean NOT NULL DEFAULT false,
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `apps_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `apps_id_unique` UNIQUE(`id`),
	CONSTRAINT `apps_project_slug_idx` UNIQUE(`project_id`,`slug`)
);

CREATE INDEX `apps_workspace_idx` ON `apps` (`workspace_id`);

