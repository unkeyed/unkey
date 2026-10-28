CREATE TABLE `ratelimit_state` (
	`fingerprint` binary(32) NOT NULL,
	`period_start_ms` bigint NOT NULL,
	`pod_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`scaled_sum` double NOT NULL,
	`delete_after_ms` bigint NOT NULL,
	CONSTRAINT `ratelimit_state_fingerprint_period_start_ms_pod_id_pk` PRIMARY KEY(`fingerprint`,`period_start_ms`,`pod_id`)
);

CREATE INDEX `ratelimit_state_delete_after_ms_idx` ON `ratelimit_state` (`delete_after_ms`);

