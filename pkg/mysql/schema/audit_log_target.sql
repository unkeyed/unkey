CREATE TABLE `audit_log_target` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`bucket_id` varchar(256) NOT NULL,
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	`audit_log_id` varchar(256) NOT NULL,
	`display_name` varchar(256) NOT NULL,
	`type` varchar(256) NOT NULL,
	`id` varchar(256) NOT NULL,
	`name` varchar(256),
	`meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_target_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_id_per_log` UNIQUE(`audit_log_id`,`id`)
);

CREATE INDEX `bucket` ON `audit_log_target` (`bucket`);

CREATE INDEX `id_idx` ON `audit_log_target` (`id`);

