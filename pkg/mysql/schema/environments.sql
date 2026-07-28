CREATE TABLE `environments` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`description` varchar(255) NOT NULL DEFAULT '',
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `environments_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `environments_id_unique` UNIQUE(`id`),
	CONSTRAINT `environments_app_slug_idx` UNIQUE(`app_id`,`slug`)
);

CREATE INDEX `environments_project_idx` ON `environments` (`project_id`);

