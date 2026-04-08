CREATE TABLE `vercel_integrations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`team_id` varchar(256),
	`access_token` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `vercel_integrations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `vercel_integrations_id_unique` UNIQUE(`id`)
);

