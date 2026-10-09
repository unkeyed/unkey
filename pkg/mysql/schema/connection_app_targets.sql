CREATE TABLE `connection_app_targets` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`connection_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`selection_mode` enum('automatic','environment','deployment') NOT NULL,
	`target_environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`target_deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	CONSTRAINT `connection_app_targets_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `connection_app_targets_connection_idx` UNIQUE(`connection_id`)
);

CREATE INDEX `connection_app_targets_deployment_idx` ON `connection_app_targets` (`target_deployment_id`);

