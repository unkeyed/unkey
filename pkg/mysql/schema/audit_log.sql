CREATE TABLE `audit_log` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	`bucket_id` varchar(256) NOT NULL,
	`event` varchar(256) NOT NULL,
	`time` bigint NOT NULL,
	`display` varchar(256) NOT NULL,
	`remote_ip` varchar(256),
	`user_agent` varchar(256),
	`actor_type` varchar(256) NOT NULL,
	`actor_id` varchar(256) NOT NULL,
	`actor_name` varchar(256),
	`actor_meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `audit_log_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_id_idx` ON `audit_log` (`workspace_id`);

CREATE INDEX `bucket_id_idx` ON `audit_log` (`bucket_id`);

CREATE INDEX `bucket_idx` ON `audit_log` (`bucket`);

CREATE INDEX `event_idx` ON `audit_log` (`event`);

CREATE INDEX `actor_id_idx` ON `audit_log` (`actor_id`);

CREATE INDEX `time_idx` ON `audit_log` (`time`);

