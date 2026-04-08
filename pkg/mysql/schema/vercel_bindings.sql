CREATE TABLE `vercel_bindings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`integration_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`environment` enum('development','preview','production') NOT NULL,
	`resource_id` varchar(256) NOT NULL,
	`resource_type` enum('rootKey','apiId') NOT NULL,
	`vercel_env_id` varchar(256) NOT NULL,
	`last_edited_by` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `vercel_bindings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `vercel_bindings_id_unique` UNIQUE(`id`),
	CONSTRAINT `project_environment_resource_type_idx` UNIQUE(`project_id`,`environment`,`resource_type`)
);

