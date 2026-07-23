CREATE TABLE `identities` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`external_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(64) NOT NULL DEFAULT '',
	`environment` varchar(256) NOT NULL DEFAULT 'default',
	`meta` json,
	`deleted` boolean NOT NULL DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `identities_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `identities_id_unique` UNIQUE(`id`),
	CONSTRAINT `project_id_external_id_deleted_idx` UNIQUE(`project_id`,`external_id`,`deleted`)
);

CREATE INDEX `identity_project_id_idx` ON `identities` (`project_id`);

