CREATE TABLE `apis` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`ip_whitelist` varchar(512),
	`auth_type` enum('key','jwt'),
	`key_auth_id` varchar(256),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`delete_protection` boolean DEFAULT false,
	CONSTRAINT `apis_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `apis_id_unique` UNIQUE(`id`),
	CONSTRAINT `apis_key_auth_id_unique` UNIQUE(`key_auth_id`)
);

CREATE INDEX `workspace_id_idx` ON `apis` (`workspace_id`);

