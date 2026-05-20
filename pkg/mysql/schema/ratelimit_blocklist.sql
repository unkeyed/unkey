CREATE TABLE `ratelimit_blocklist` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(191) NOT NULL,
	`namespace` varchar(255) NOT NULL,
	`identifier` varchar(255) NOT NULL,
	`duration_ms` bigint unsigned NOT NULL,
	`sequence` bigint NOT NULL,
	`limit` bigint unsigned NOT NULL,
	`expires_at` bigint unsigned NOT NULL,
	CONSTRAINT `ratelimit_blocklist_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_propagation_key` UNIQUE(`workspace_id`,`namespace`,`identifier`,`duration_ms`,`sequence`)
);

CREATE INDEX `expires_at_idx` ON `ratelimit_blocklist` (`expires_at`);

