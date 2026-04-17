CREATE TABLE `portal_configurations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64),
	`key_auth_id` varchar(64),
	`enabled` boolean NOT NULL DEFAULT true,
	`return_url` varchar(500),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `portal_configurations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `portal_configurations_id_unique` UNIQUE(`id`),
	CONSTRAINT `idx_app_id` UNIQUE(`app_id`),
	CONSTRAINT `idx_key_auth_id` UNIQUE(`key_auth_id`)
);

CREATE INDEX `idx_workspace` ON `portal_configurations` (`workspace_id`);

