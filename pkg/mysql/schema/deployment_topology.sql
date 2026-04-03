CREATE TABLE `deployment_topology` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(64) NOT NULL,
	`deployment_id` varchar(64) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`autoscaling_replicas_min` int unsigned NOT NULL DEFAULT 1,
	`autoscaling_replicas_max` int unsigned NOT NULL DEFAULT 1,
	`autoscaling_threshold_cpu` tinyint unsigned,
	`autoscaling_threshold_memory` tinyint unsigned,
	`vpa_update_mode` enum('off','initial','recreate','in_place_or_recreate'),
	`vpa_controlled_resources` enum('cpu','memory','both'),
	`vpa_controlled_values` enum('requests','requests_and_limits'),
	`vpa_cpu_min_millicores` int unsigned,
	`vpa_cpu_max_millicores` int unsigned,
	`vpa_memory_min_mib` int unsigned,
	`vpa_memory_max_mib` int unsigned,
	`desired_status` enum('stopped','running') NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployment_topology_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_region_per_deployment` UNIQUE(`deployment_id`,`region_id`)
);

CREATE INDEX `workspace_idx` ON `deployment_topology` (`workspace_id`);

CREATE INDEX `status_idx` ON `deployment_topology` (`desired_status`);

