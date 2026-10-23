CREATE TABLE `ratelimit_gcra` (
	`name` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`writer_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`spent_ns` bigint NOT NULL,
	`fresh_start_ns` bigint NOT NULL,
	`expires_at_ns` bigint NOT NULL,
	`updated_at_ms` bigint NOT NULL,
	CONSTRAINT `ratelimit_gcra_name_writer_id_pk` PRIMARY KEY(`name`,`writer_id`)
);

CREATE INDEX `ratelimit_gcra_expires_at_ns_idx` ON `ratelimit_gcra` (`expires_at_ns`);

