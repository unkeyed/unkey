CREATE TABLE `cilium_network_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`project_id` varchar(255) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(255) NOT NULL,
	`deployment_id` varchar(128) NOT NULL,
	`k8s_name` varchar(64) NOT NULL,
	`k8s_namespace` varchar(255) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`policy` json NOT NULL,
	`version` bigint unsigned NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `cilium_network_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `cilium_network_policies_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_version_per_region` UNIQUE(`region_id`,`version`)
);

CREATE INDEX `idx_deployment_region` ON `cilium_network_policies` (`deployment_id`,`region_id`);

