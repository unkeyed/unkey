CREATE TABLE `github_repo_connections` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`project_id` varchar(64) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`installation_id` bigint NOT NULL,
	`repository_id` bigint NOT NULL,
	`repository_full_name` varchar(500) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `github_repo_connections_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `github_repo_connections_app_id_unique` UNIQUE(`app_id`)
);

CREATE INDEX `installation_id_idx` ON `github_repo_connections` (`installation_id`);

