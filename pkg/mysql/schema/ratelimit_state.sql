CREATE TABLE `ratelimit_state` (
	`fingerprint` binary(32) NOT NULL,
	`period_start_ns` bigint NOT NULL,
	`pod_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`scaled_sum` double NOT NULL,
	`delete_after_ns` bigint NOT NULL,
	CONSTRAINT `ratelimit_state_fingerprint_period_start_ns_pod_id_pk` PRIMARY KEY(`fingerprint`,`period_start_ns`,`pod_id`)
);

CREATE INDEX `ratelimit_state_delete_after_ns_idx` ON `ratelimit_state` (`delete_after_ns`);

