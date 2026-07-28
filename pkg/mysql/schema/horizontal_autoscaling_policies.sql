CREATE TABLE `horizontal_autoscaling_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`replicas_min` int NOT NULL,
	`replicas_max` int NOT NULL,
	`memory_threshold` tinyint,
	`cpu_threshold` tinyint,
	`rps_threshold` tinyint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `horizontal_autoscaling_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `horizontal_autoscaling_policies_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_idx` ON `horizontal_autoscaling_policies` (`workspace_id`);

