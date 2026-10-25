CREATE TABLE `notification_subscriptions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(128) NOT NULL,
	`notification_type` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`enabled` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notification_subscriptions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notification_subscriptions_id_unique` UNIQUE(`id`)
);

CREATE INDEX `notification_subscriptions_routing_idx` ON `notification_subscriptions` (`workspace_id`,`notification_type`,`enabled`);

