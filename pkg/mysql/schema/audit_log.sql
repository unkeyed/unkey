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

CREATE INDEX `workspace_id_bucket_time_idx` ON `audit_log` (`workspace_id`,`bucket`,`time`);

