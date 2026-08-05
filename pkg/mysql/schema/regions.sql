CREATE TABLE `regions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(64) NOT NULL,
	`platform` varchar(64) NOT NULL,
	`can_schedule` boolean NOT NULL DEFAULT true,
	CONSTRAINT `regions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `regions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_region_per_platform` UNIQUE(`name`,`platform`)
);

