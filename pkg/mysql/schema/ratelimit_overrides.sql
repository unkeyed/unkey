CREATE TABLE `ratelimit_overrides` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`namespace_id` varchar(256) NOT NULL,
	`identifier` varchar(512) NOT NULL,
	`limit` bigint unsigned NOT NULL,
	`duration` bigint unsigned NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_overrides_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimit_overrides_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_identifier_per_namespace_idx` UNIQUE(`namespace_id`,`identifier`)
);

