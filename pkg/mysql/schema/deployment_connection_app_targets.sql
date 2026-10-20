CREATE TABLE `deployment_connection_app_targets` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`connection_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`selection_mode` enum('automatic','environment','deployment') NOT NULL,
	`target_environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`target_deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	CONSTRAINT `deployment_connection_app_targets_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `deployment_connection_app_targets_deployment_connection_idx` UNIQUE(`deployment_id`,`connection_id`)
);

CREATE INDEX `deployment_connection_app_targets_target_deployment_idx` ON `deployment_connection_app_targets` (`target_deployment_id`);

