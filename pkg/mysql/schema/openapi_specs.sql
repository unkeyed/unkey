CREATE TABLE `openapi_specs` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`portal_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`content` longblob NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `openapi_specs_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `openapi_specs_id_unique` UNIQUE(`id`),
	CONSTRAINT `idx_openapi_specs_on_deployment_id` UNIQUE(`deployment_id`),
	CONSTRAINT `workspace_deployment_idx` UNIQUE(`workspace_id`,`deployment_id`),
	CONSTRAINT `workspace_portal_idx` UNIQUE(`workspace_id`,`portal_id`)
);

