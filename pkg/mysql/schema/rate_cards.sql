CREATE TABLE `rate_cards` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`currency` varchar(3) NOT NULL DEFAULT 'USD',
	`config` json NOT NULL,
	`selectable` boolean NOT NULL DEFAULT false,
	`archived` boolean NOT NULL DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `rate_cards_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `rate_cards_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_id_name_idx` UNIQUE(`workspace_id`,`name`)
);

