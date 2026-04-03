CREATE TABLE `custom_domains` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`environment_id` varchar(256) NOT NULL,
	`domain` varchar(256) NOT NULL,
	`challenge_type` enum('HTTP-01','DNS-01') NOT NULL,
	`verification_status` enum('pending','verifying','verified','failed') NOT NULL DEFAULT 'pending',
	`verification_token` varchar(64) NOT NULL,
	`ownership_verified` boolean NOT NULL DEFAULT false,
	`cname_verified` boolean NOT NULL DEFAULT false,
	`target_cname` varchar(256) NOT NULL,
	`last_checked_at` bigint,
	`check_attempts` int NOT NULL DEFAULT 0,
	`verification_error` varchar(512),
	`invocation_id` varchar(256),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `custom_domains_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `custom_domains_id_unique` UNIQUE(`id`),
	CONSTRAINT `custom_domains_target_cname_unique` UNIQUE(`target_cname`),
	CONSTRAINT `unique_domain_workspace_idx` UNIQUE(`workspace_id`,`domain`)
);

CREATE INDEX `project_idx` ON `custom_domains` (`project_id`);

CREATE INDEX `verification_status_idx` ON `custom_domains` (`verification_status`);

