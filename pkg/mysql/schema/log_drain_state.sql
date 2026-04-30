CREATE TABLE `log_drain_state` (
	`drain_id` varchar(64) NOT NULL,
	`last_delivery_at` bigint,
	`last_attempt_at` bigint,
	`last_error` varchar(1024),
	`consecutive_failures` int NOT NULL DEFAULT 0,
	`paused_reason` varchar(256),
	`total_records_delivered` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint NOT NULL,
	CONSTRAINT `log_drain_state_drain_id` PRIMARY KEY(`drain_id`)
);

