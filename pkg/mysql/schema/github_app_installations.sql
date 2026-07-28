CREATE TABLE `github_app_installations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`installation_id` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `github_app_installations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_installation_idx` UNIQUE(`workspace_id`,`installation_id`)
);

