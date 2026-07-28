CREATE TABLE `workspaces` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`org_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(256) NOT NULL,
	`slug` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`k8s_namespace` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`beta_features` json NOT NULL,
	`subscriptions` json,
	`enabled` boolean NOT NULL DEFAULT true,
	`delete_protection` boolean DEFAULT false,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `workspaces_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspaces_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspaces_org_id_unique` UNIQUE(`org_id`),
	CONSTRAINT `workspaces_slug_unique` UNIQUE(`slug`),
	CONSTRAINT `workspaces_k8s_namespace_unique` UNIQUE(`k8s_namespace`)
);

