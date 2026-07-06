CREATE TABLE `billing_period_rate_cards` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`identity_id` varchar(256) NOT NULL,
	`year` int NOT NULL,
	`month` int NOT NULL,
	`rate_card_id` varchar(256) NOT NULL,
	`resolved_from` enum('selection','assignment','workspace_default') NOT NULL,
	`pushed_at` bigint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `billing_period_rate_cards_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `billing_period_rate_cards_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_identity_period_idx` UNIQUE(`workspace_id`,`identity_id`,`year`,`month`)
);

