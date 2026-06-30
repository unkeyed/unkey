CREATE TABLE `shared_secrets` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`expires_at` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`encrypted` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	CONSTRAINT `shared_secrets_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `shared_secrets_id_unique` UNIQUE(`id`)
);

CREATE INDEX `expires_at_idx` ON `shared_secrets` (`expires_at`);

