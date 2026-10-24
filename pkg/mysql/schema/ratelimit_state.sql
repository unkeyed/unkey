CREATE TABLE `ratelimit_state` (
	`fingerprint` binary(32) NOT NULL,
	`session_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`anchor_at_ns` bigint NOT NULL,
	`consumed_ns` bigint NOT NULL,
	`compact_after_ms` bigint NOT NULL,
	CONSTRAINT `ratelimit_state_fingerprint_session_id_pk` PRIMARY KEY(`fingerprint`,`session_id`)
);

CREATE INDEX `ratelimit_state_compact_after_ms_idx` ON `ratelimit_state` (`compact_after_ms`);

