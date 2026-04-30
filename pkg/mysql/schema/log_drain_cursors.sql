CREATE TABLE `log_drain_cursors` (
	`drain_id` varchar(64) NOT NULL,
	`group_key` varchar(128) NOT NULL,
	`time_ms` bigint NOT NULL,
	`last_id` varchar(256) NOT NULL DEFAULT '',
	`blocked` boolean NOT NULL DEFAULT false,
	`blocked_reason` varchar(256),
	`updated_at` bigint NOT NULL,
	CONSTRAINT `log_drain_cursors_drain_id_group_key_pk` PRIMARY KEY(`drain_id`,`group_key`)
);
