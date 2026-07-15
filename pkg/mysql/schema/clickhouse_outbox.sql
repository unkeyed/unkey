CREATE TABLE `clickhouse_outbox` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`version` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`event_id` varchar(256) NOT NULL,
	`payload` json NOT NULL,
	`created_at` bigint NOT NULL,
	`deleted_at` bigint unsigned,
	CONSTRAINT `clickhouse_outbox_pk` PRIMARY KEY(`pk`)
);

CREATE INDEX `drainer_pending_idx` ON `clickhouse_outbox` (`deleted_at`,`version`,`pk`);

