CREATE TABLE `ratelimit_state` (
	`fingerprint` binary(32) NOT NULL,
	`bucket_start_ns` bigint NOT NULL,
	`pod_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`consumed_ns` bigint NOT NULL,
	`reset_at_ns` bigint NOT NULL,
	`compact_after_ns` bigint NOT NULL,
	CONSTRAINT `ratelimit_state_fingerprint_bucket_start_ns_pod_id_pk` PRIMARY KEY(`fingerprint`,`bucket_start_ns`,`pod_id`)
);

CREATE INDEX `ratelimit_state_compact_after_ns_idx` ON `ratelimit_state` (`compact_after_ns`);

