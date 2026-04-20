CREATE TABLE `sentinel_tiers` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`tier_id` varchar(64) NOT NULL,
	`version` varchar(32) NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`price_per_second` decimal(12,8) NOT NULL,
	`effective_from` bigint NOT NULL,
	`effective_until` bigint,
	CONSTRAINT `sentinel_tiers_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `sentinel_tiers_id_unique` UNIQUE(`id`),
	CONSTRAINT `sentinel_tiers_tier_version_unique` UNIQUE(`tier_id`,`version`)
);

