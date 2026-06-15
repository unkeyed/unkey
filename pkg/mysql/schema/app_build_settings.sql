CREATE TABLE `app_build_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(128) NOT NULL,
	`dockerfile` varchar(500),
	`docker_context` varchar(500) NOT NULL DEFAULT '.',
	`watch_paths` json NOT NULL DEFAULT ('[]'),
	`auto_deploy` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_build_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_build_settings_app_env_idx` UNIQUE(`app_id`,`environment_id`)
);

