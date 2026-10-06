CREATE TABLE `agent_signups` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`agent_registration_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workos_user_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`root_key_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`status` varchar(32) NOT NULL,
	`requested_permissions` json,
	`created_at_m` bigint NOT NULL,
	`updated_at_m` bigint,
	CONSTRAINT `agent_signups_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `agent_signups_id_unique` UNIQUE(`id`),
	CONSTRAINT `agent_signups_agent_registration_id_unique` UNIQUE(`agent_registration_id`)
);
