CREATE TABLE `flag_overrides` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`flag_slug` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`value` boolean NOT NULL,
	CONSTRAINT `flag_overrides_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_flag_idx` UNIQUE(`workspace_id`,`flag_slug`)
);

