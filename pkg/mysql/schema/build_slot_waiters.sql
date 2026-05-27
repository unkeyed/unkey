CREATE TABLE `build_slot_waiters` (
	`deployment_id` varchar(255) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`awakeable_id` varchar(255) NOT NULL,
	`is_production` boolean NOT NULL,
	`enqueued_at` bigint NOT NULL,
	CONSTRAINT `build_slot_waiters_deployment_id` PRIMARY KEY(`deployment_id`)
);

CREATE INDEX `idx_workspace_priority` ON `build_slot_waiters` (`workspace_id`,`is_production`,`enqueued_at`);

