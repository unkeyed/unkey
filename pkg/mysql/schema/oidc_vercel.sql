CREATE TABLE `oidc_vercel` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(256) NOT NULL,
	`audience` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`vercel_team_slug` varchar(100) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`vercel_project_name` varchar(100) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`vercel_environment` varchar(100) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`enabled` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `oidc_vercel_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `oidc_vercel_id_unique` UNIQUE(`id`),
	CONSTRAINT `oidc_vercel_audience_unique` UNIQUE(`audience`),
	CONSTRAINT `workspace_vercel_team_project_environment_idx` UNIQUE(`workspace_id`,`vercel_team_slug`,`vercel_project_name`,`vercel_environment`)
);

CREATE INDEX `workspace_id_idx` ON `oidc_vercel` (`workspace_id`);
