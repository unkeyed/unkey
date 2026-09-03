CREATE TABLE `alert_events` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`metric` enum('error_5xx','error_4xx','requests','requests_drop','egress_bytes','cpu_seconds','memory_utilization','oom_killed','crash_loop') NOT NULL,
	`status` enum('open','resolved') NOT NULL DEFAULT 'open',
	`fired_at` bigint NOT NULL,
	`last_seen_at` bigint NOT NULL,
	`resolved_at` bigint,
	`resolution_message` varchar(1000),
	`observed_value` double NOT NULL,
	`baseline_mean` double NOT NULL,
	`baseline_stddev` double NOT NULL,
	`threshold_sigma` double NOT NULL,
	`window_start` bigint NOT NULL,
	`window_end` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `alert_events_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `alert_events_id_unique` UNIQUE(`id`)
);

CREATE INDEX `status_idx` ON `alert_events` (`status`);

CREATE INDEX `workspace_status_fired_at_idx` ON `alert_events` (`workspace_id`,`status`,`fired_at`);

CREATE INDEX `workspace_app_environment_status_idx` ON `alert_events` (`workspace_id`,`app_id`,`environment_id`,`status`);
