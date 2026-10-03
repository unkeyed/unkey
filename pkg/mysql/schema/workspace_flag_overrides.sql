CREATE TABLE `workspace_flag_overrides` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`flag_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`value` json NOT NULL,
	CONSTRAINT `workspace_flag_overrides_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_flag_idx` UNIQUE(`workspace_id`,`flag_id`)
);

