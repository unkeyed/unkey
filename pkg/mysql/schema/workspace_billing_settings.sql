CREATE TABLE `workspace_billing_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`default_rate_card_id` varchar(256),
	`stripe_connect_encrypted` text,
	`stripe_connect_encryption_key_id` varchar(256),
	`stripe_connect_status` enum('pending','linked'),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `workspace_billing_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_billing_settings_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_billing_settings_workspace_id_unique` UNIQUE(`workspace_id`)
);

