CREATE TABLE `logdrains` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(128) NOT NULL,
	`stream` enum('audit_logs') NOT NULL,
	`config` longblob NOT NULL,
	`enabled` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `logdrains_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `logdrains_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_id_idx` ON `logdrains` (`workspace_id`);
