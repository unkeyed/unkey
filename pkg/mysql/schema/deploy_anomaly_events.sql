CREATE TABLE `deploy_anomaly_events` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`metric` enum('oom_killed','crash_loop') NOT NULL,
	`event_time` bigint NOT NULL,
	`received_at` bigint NOT NULL,
	`processed_at` bigint,
	CONSTRAINT `deploy_anomaly_events_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `deploy_anomaly_events_id_unique` UNIQUE(`id`)
);

CREATE INDEX `deploy_anomaly_events_pending_idx` ON `deploy_anomaly_events` (`workspace_id`,`processed_at`,`pk`);
