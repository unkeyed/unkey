CREATE TABLE `ratelimit_namespaces` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(33) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(32) COLLATE utf8mb4_0900_as_cs NOT NULL DEFAULT '',
	`name` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_namespaces_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimit_namespaces_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`workspace_id`,`name`)
);

CREATE INDEX `ratelimit_namespaces_project_id_idx` ON `ratelimit_namespaces` (`project_id`);

