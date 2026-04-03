CREATE TABLE `keys_roles` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`key_id` varchar(256) NOT NULL,
	`role_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_roles_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_roles_role_id_key_id_workspace_id` UNIQUE(`role_id`,`key_id`,`workspace_id`),
	CONSTRAINT `unique_key_id_role_id` UNIQUE(`key_id`,`role_id`)
);

