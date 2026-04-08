CREATE TABLE `openapi_specs` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`deployment_id` varchar(128),
	`portal_config_id` varchar(256),
	`content` longblob NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `openapi_specs_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `openapi_specs_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_deployment_idx` UNIQUE(`workspace_id`,`deployment_id`),
	CONSTRAINT `workspace_portal_config_idx` UNIQUE(`workspace_id`,`portal_config_id`)
);

