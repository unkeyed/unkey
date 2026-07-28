CREATE TABLE `roles_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`role_id` varchar(256) NOT NULL,
	`permission_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `roles_permissions_role_id_permission_id_workspace_id` UNIQUE(`role_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `unique_tuple_permission_id_role_id` UNIQUE(`permission_id`,`role_id`)
);

