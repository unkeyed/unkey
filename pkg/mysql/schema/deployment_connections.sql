CREATE TABLE `deployment_connections` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`connection_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`resource_type` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`resource_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(63) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `deployment_connections_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `deployment_connections_deployment_connection_idx` UNIQUE(`deployment_id`,`connection_id`),
	CONSTRAINT `deployment_connections_deployment_name_idx` UNIQUE(`deployment_id`,`name`)
);

CREATE INDEX `deployment_connections_connection_idx` ON `deployment_connections` (`connection_id`);

CREATE INDEX `deployment_connections_app_env_idx` ON `deployment_connections` (`app_id`,`environment_id`);

