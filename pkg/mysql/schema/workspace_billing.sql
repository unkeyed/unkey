CREATE TABLE `workspace_billing` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`tier` varchar(256) DEFAULT 'Free',
	`stripe_customer_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`plan` varchar(64),
	`plan_override` varchar(64),
	`spend_budget_cents` bigint unsigned,
	`spend_budget_stop` boolean NOT NULL DEFAULT false,
	`spend_suspended` boolean NOT NULL DEFAULT false,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `workspace_billing_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_billing_workspace_id_unique` UNIQUE(`workspace_id`)
);

CREATE INDEX `spend_budget_cents_idx` ON `workspace_billing` (`spend_budget_cents`);

CREATE INDEX `spend_suspended_idx` ON `workspace_billing` (`spend_suspended`);

