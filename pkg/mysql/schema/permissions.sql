CREATE TABLE `permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`slug` varchar(128) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `permissions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_slug_per_workspace_idx` UNIQUE(`workspace_id`,`slug`)
);

