CREATE TABLE `instances` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`deployment_id` varchar(255) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`project_id` varchar(255) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`k8s_name` varchar(255) NOT NULL,
	`address` varchar(255) NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`storage_mib` int unsigned NOT NULL DEFAULT 0,
	`status` enum('inactive','pending','running','failed') NOT NULL,
	CONSTRAINT `instances_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `instances_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_address_per_region` UNIQUE(`address`,`region_id`),
	CONSTRAINT `unique_k8s_name_per_region` UNIQUE(`k8s_name`,`region_id`)
);

CREATE INDEX `idx_deployment_id` ON `instances` (`deployment_id`);

CREATE INDEX `idx_region` ON `instances` (`region_id`);

