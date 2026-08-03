CREATE TABLE `clusters` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`cell_id` varchar(64) COLLATE utf8mb4_0900_as_cs,
	`region_id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`last_heartbeat_at` bigint unsigned NOT NULL,
	CONSTRAINT `clusters_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `clusters_id_unique` UNIQUE(`id`),
	CONSTRAINT `clusters_cell_id_unique` UNIQUE(`cell_id`),
	CONSTRAINT `clusters_region_id_unique` UNIQUE(`region_id`)
);
