CREATE TABLE `certificates` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`hostname` varchar(255) NOT NULL,
	`certificate` text NOT NULL,
	`encrypted_private_key` text NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `certificates_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `certificates_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_hostname` UNIQUE(`hostname`)
);

