CREATE TABLE `billing_subscriptions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`product` enum('api','compute') NOT NULL,
	`stripe_subscription_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	CONSTRAINT `billing_subscriptions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `billing_subscriptions_stripe_subscription_id_unique` UNIQUE(`stripe_subscription_id`),
	CONSTRAINT `unique_product_per_workspace` UNIQUE(`workspace_id`,`product`)
);

CREATE INDEX `workspace_id_idx` ON `billing_subscriptions` (`workspace_id`);

