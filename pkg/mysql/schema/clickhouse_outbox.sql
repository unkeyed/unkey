CREATE TABLE `clickhouse_outbox` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`version` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`event_id` varchar(256) NOT NULL,
	`payload` json NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `clickhouse_outbox_pk` PRIMARY KEY(`pk`)
);

