CREATE TABLE `deployment_topology` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(64) NOT NULL,
	`deployment_id` varchar(64) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`autoscaling_replicas_min` int unsigned NOT NULL DEFAULT 1,
	`autoscaling_replicas_max` int unsigned NOT NULL DEFAULT 1,
	`autoscaling_threshold_cpu` tinyint unsigned,
	`autoscaling_threshold_memory` tinyint unsigned,
	`version` bigint unsigned NOT NULL,
	`desired_status` enum('stopped','running') NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployment_topology_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_region_per_deployment` UNIQUE(`deployment_id`,`region_id`),
	CONSTRAINT `unique_version_per_region` UNIQUE(`region_id`,`version`)
);

CREATE INDEX `workspace_idx` ON `deployment_topology` (`workspace_id`);

CREATE INDEX `status_idx` ON `deployment_topology` (`desired_status`);

