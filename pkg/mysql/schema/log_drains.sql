CREATE TABLE `log_drains` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256),
	`name` varchar(256) NOT NULL,
	`provider` enum('axiom') NOT NULL,
	`config` json NOT NULL,
	`sources` json NOT NULL DEFAULT ('["runtime","request"]'),
	`environments` json NOT NULL DEFAULT ('["production"]'),
	`apps` json NOT NULL DEFAULT ('[]'),
	`filters` json NOT NULL DEFAULT ('{}'),
	`delivery_mode` enum('batch','stream') NOT NULL DEFAULT 'batch',
	`enabled` boolean NOT NULL DEFAULT true,
	`extension_installation_id` varchar(128),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`deleted_at` bigint,
	CONSTRAINT `log_drains_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `log_drains_id_unique` UNIQUE(`id`)
);

CREATE INDEX `log_drains_project_idx` ON `log_drains` (`project_id`,`deleted_at`);

CREATE INDEX `log_drains_workspace_idx` ON `log_drains` (`workspace_id`,`deleted_at`);

CREATE INDEX `log_drains_extension_installation_idx` ON `log_drains` (`extension_installation_id`);

