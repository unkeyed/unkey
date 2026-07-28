CREATE TABLE `app_build_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`dockerfile` varchar(500) COLLATE utf8mb4_0900_as_cs,
	`docker_context` varchar(500) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '.',
	`build_command` varchar(1000) COLLATE utf8mb4_0900_as_cs,
	`watch_paths` json NOT NULL DEFAULT ('[]'),
	`auto_deploy` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_build_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_build_settings_app_env_idx` UNIQUE(`app_id`,`environment_id`)
);

