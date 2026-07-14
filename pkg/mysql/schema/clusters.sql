CREATE TABLE `clusters` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`platform` varchar(64) NOT NULL DEFAULT '',
	`region` varchar(64) NOT NULL DEFAULT '',
	`state` enum('active','disabled') NOT NULL DEFAULT 'disabled',
	`last_heartbeat_at` bigint unsigned NOT NULL,
	CONSTRAINT `clusters_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `clusters_id_unique` UNIQUE(`id`),
	CONSTRAINT `clusters_region_id_unique` UNIQUE(`region_id`)
);

