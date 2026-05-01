CREATE TABLE `ratelimit_window_counts` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(191) NOT NULL,
	`namespace` varchar(255) NOT NULL,
	`identifier` varchar(255) NOT NULL,
	`duration_ms` bigint unsigned NOT NULL,
	`sequence` bigint NOT NULL,
	`region` varchar(48) NOT NULL,
	`count` bigint unsigned NOT NULL,
	`expires_at` bigint unsigned NOT NULL,
	`updated_at` bigint unsigned NOT NULL,
	CONSTRAINT `ratelimit_window_counts_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_window_region` UNIQUE(`workspace_id`,`namespace`,`identifier`,`duration_ms`,`sequence`,`region`)
);

CREATE INDEX `expires_at_idx` ON `ratelimit_window_counts` (`expires_at`);

CREATE INDEX `lookup_idx` ON `ratelimit_window_counts` (`workspace_id`,`namespace`,`identifier`,`duration_ms`,`sequence`);

