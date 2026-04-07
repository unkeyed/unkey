CREATE TABLE `app_runtime_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(128) NOT NULL,
	`port` int NOT NULL DEFAULT 8080,
	`cpu_millicores` int NOT NULL DEFAULT 250,
	`memory_mib` int NOT NULL DEFAULT 256,
	`command` json NOT NULL DEFAULT ('[]'),
	`healthcheck` json,
	`shutdown_signal` enum('SIGTERM','SIGINT','SIGQUIT','SIGKILL') NOT NULL DEFAULT 'SIGTERM',
	`sentinel_config` longblob NOT NULL,
	`openapi_spec_path` varchar(512),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_runtime_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_runtime_settings_app_env_idx` UNIQUE(`app_id`,`environment_id`)
);

