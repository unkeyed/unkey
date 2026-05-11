CREATE TABLE `extension_installations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`extension_slug` varchar(128) NOT NULL,
	`instance_name` varchar(256) NOT NULL,
	`config` json NOT NULL,
	`status` enum('active','degraded','disabled','verifying','failed') NOT NULL DEFAULT 'active',
	`oauth_connected` boolean NOT NULL DEFAULT false,
	`last_event_at` bigint,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	`deleted_at` bigint,
	CONSTRAINT `extension_installations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `extension_installations_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_project_extension_instance_idx` UNIQUE(`project_id`,`extension_slug`,`instance_name`)
);

CREATE INDEX `workspace_idx` ON `extension_installations` (`workspace_id`);

CREATE INDEX `project_idx` ON `extension_installations` (`project_id`);

CREATE INDEX `extension_slug_idx` ON `extension_installations` (`extension_slug`);
