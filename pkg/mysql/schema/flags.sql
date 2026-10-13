CREATE TABLE `flags` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`slug` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`description` varchar(1024) NOT NULL,
	`default_enabled` boolean NOT NULL,
	`allow_opt_in` boolean NOT NULL DEFAULT false,
	`allow_opt_out` boolean NOT NULL DEFAULT false,
	CONSTRAINT `flags_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `flags_slug_unique` UNIQUE(`slug`)
);

