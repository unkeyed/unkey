CREATE TABLE `frontline_routes` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`project_id` varchar(255) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`deployment_id` varchar(255) NOT NULL,
	`environment_id` varchar(255) NOT NULL,
	`fully_qualified_domain_name` varchar(256) NOT NULL,
	`sticky` enum('none','branch','environment','live','deployment') NOT NULL DEFAULT 'none',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `frontline_routes_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `frontline_routes_id_unique` UNIQUE(`id`),
	CONSTRAINT `frontline_routes_fully_qualified_domain_name_unique` UNIQUE(`fully_qualified_domain_name`)
);

CREATE INDEX `project_id_idx` ON `frontline_routes` (`project_id`);

CREATE INDEX `environment_id_idx` ON `frontline_routes` (`environment_id`);

CREATE INDEX `deployment_id_idx` ON `frontline_routes` (`deployment_id`);

CREATE INDEX `fqdn_environment_deployment_idx` ON `frontline_routes` (`fully_qualified_domain_name`,`environment_id`,`deployment_id`);

