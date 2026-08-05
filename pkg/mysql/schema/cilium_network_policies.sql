CREATE TABLE `cilium_network_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(255) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`deployment_id` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`k8s_name` varchar(64) NOT NULL,
	`k8s_namespace` varchar(255) NOT NULL,
	`region_id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`policy` json NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `cilium_network_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `cilium_network_policies_id_unique` UNIQUE(`id`)
);

CREATE INDEX `idx_deployment_region` ON `cilium_network_policies` (`deployment_id`,`region_id`);

