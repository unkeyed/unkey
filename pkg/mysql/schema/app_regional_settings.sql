CREATE TABLE `app_regional_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(128) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`replicas` int NOT NULL DEFAULT 1,
	`horizontal_autoscaling_policy_id` varchar(64),
	`vertical_autoscaling_policy_id` varchar(64),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_regional_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_app_env_region` UNIQUE(`app_id`,`environment_id`,`region_id`)
);

CREATE INDEX `workspace_idx` ON `app_regional_settings` (`workspace_id`);

