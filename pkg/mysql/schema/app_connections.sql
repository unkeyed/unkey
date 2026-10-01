CREATE TABLE `app_connections` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`resource_type` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`resource_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(63) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`selection_mode` enum('automatic','environment','deployment'),
	`target_environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`target_deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_connections_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_connections_id_unique` UNIQUE(`id`),
	CONSTRAINT `app_connections_app_name_idx` UNIQUE(`app_id`,`environment_id`,`name`),
	CONSTRAINT `app_connections_app_resource_idx` UNIQUE(`app_id`,`environment_id`,`resource_type`,`resource_id`)
);

CREATE INDEX `app_connections_project_idx` ON `app_connections` (`project_id`);

CREATE INDEX `app_connections_resource_idx` ON `app_connections` (`resource_type`,`resource_id`);

CREATE INDEX `app_connections_workspace_idx` ON `app_connections` (`workspace_id`,`resource_type`);

CREATE INDEX `app_connections_target_deployment_idx` ON `app_connections` (`target_deployment_id`);

