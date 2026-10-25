CREATE TABLE `notification_destinations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(128) NOT NULL,
	`channel` enum('discord','slack','webhook') NOT NULL,
	`config` longblob NOT NULL,
	`enabled` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notification_destinations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notification_destinations_id_unique` UNIQUE(`id`)
);

CREATE INDEX `notification_destinations_workspace_idx` ON `notification_destinations` (`workspace_id`);

