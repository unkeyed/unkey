CREATE TABLE `app_docker_sources` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`image` varchar(512) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_docker_sources_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_docker_sources_app_id_unique` UNIQUE(`app_id`)
);

