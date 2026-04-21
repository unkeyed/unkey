CREATE TABLE `sentinels` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`project_id` varchar(255) NOT NULL,
	`environment_id` varchar(255) NOT NULL,
	`subscription_id` varchar(64) NOT NULL,
	`k8s_name` varchar(64) NOT NULL,
	`k8s_address` varchar(255) NOT NULL,
	`region_id` varchar(255) NOT NULL,
	`image` varchar(255) NOT NULL,
	`running_image` varchar(255) NOT NULL DEFAULT '',
	`desired_state` enum('running','standby','archived') NOT NULL DEFAULT 'running',
	`health` enum('unknown','paused','healthy','unhealthy') NOT NULL DEFAULT 'unknown',
	`desired_replicas` int NOT NULL,
	`available_replicas` int NOT NULL DEFAULT 0,
	`deploy_status` enum('idle','progressing','ready','failed') NOT NULL DEFAULT 'idle',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `sentinels_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `sentinels_id_unique` UNIQUE(`id`),
	CONSTRAINT `sentinels_k8s_name_unique` UNIQUE(`k8s_name`),
	CONSTRAINT `sentinels_k8s_address_unique` UNIQUE(`k8s_address`),
	CONSTRAINT `one_env_per_region` UNIQUE(`environment_id`,`region_id`)
);

CREATE INDEX `idx_environment_health_region_routing` ON `sentinels` (`environment_id`,`region_id`,`health`);

