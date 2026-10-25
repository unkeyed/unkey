CREATE TABLE `notification_subscription_targets` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`subscription_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`channel` enum('email','discord','slack','webhook') NOT NULL,
	`audience` int unsigned,
	`user_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`destination_id` varchar(48) COLLATE utf8mb4_0900_as_cs,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `notification_subscription_targets_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `notification_subscription_targets_id_unique` UNIQUE(`id`),
	CONSTRAINT `notification_targets_exactly_one_recipient` CHECK((`notification_subscription_targets`.`audience` IS NOT NULL) + (`notification_subscription_targets`.`user_id` IS NOT NULL) + (`notification_subscription_targets`.`destination_id` IS NOT NULL) = 1),
	CONSTRAINT `notification_targets_audience_specified` CHECK(`notification_subscription_targets`.`audience` IS NULL OR `notification_subscription_targets`.`audience` > 0),
	CONSTRAINT `notification_targets_channel_recipient` CHECK((`notification_subscription_targets`.`channel` = 'email' AND `notification_subscription_targets`.`destination_id` IS NULL) OR (`notification_subscription_targets`.`channel` <> 'email' AND `notification_subscription_targets`.`destination_id` IS NOT NULL))
);

CREATE INDEX `notification_targets_subscription_idx` ON `notification_subscription_targets` (`workspace_id`,`subscription_id`);

CREATE INDEX `notification_targets_destination_idx` ON `notification_subscription_targets` (`workspace_id`,`destination_id`);

