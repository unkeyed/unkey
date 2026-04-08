CREATE TABLE `app_environment_variables` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(128) NOT NULL,
	`key` varchar(256) NOT NULL,
	`value` varchar(4096) NOT NULL,
	`type` enum('recoverable','writeonly') NOT NULL,
	`description` varchar(255),
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_environment_variables_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_environment_variables_id_unique` UNIQUE(`id`),
	CONSTRAINT `app_env_id_key` UNIQUE(`app_id`,`environment_id`,`key`)
);

