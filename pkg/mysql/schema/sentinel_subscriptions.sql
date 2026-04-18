CREATE TABLE `sentinel_subscriptions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`sentinel_id` varchar(64) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`region_id` varchar(255) NOT NULL,
	`tier_id` varchar(64) NOT NULL,
	`tier_version` varchar(32) NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`replicas` int NOT NULL,
	`price_per_second` decimal(12,8) NOT NULL,
	`created_at` bigint NOT NULL,
	`terminated_at` bigint,
	`open_sentinel_id` varchar(64) GENERATED ALWAYS AS ((CASE WHEN `terminated_at` IS NULL THEN `sentinel_id` ELSE NULL END)) VIRTUAL,
	CONSTRAINT `sentinel_subscriptions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `sentinel_subscriptions_id_unique` UNIQUE(`id`),
	CONSTRAINT `one_open_subscription_per_sentinel` UNIQUE(`open_sentinel_id`)
);

CREATE INDEX `idx_sentinel_created` ON `sentinel_subscriptions` (`sentinel_id`,`created_at`);

CREATE INDEX `idx_workspace_period` ON `sentinel_subscriptions` (`workspace_id`,`created_at`,`terminated_at`);

