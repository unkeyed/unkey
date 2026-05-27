CREATE TABLE `build_slots` (
	`deployment_id` varchar(255) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`acquired_at` bigint NOT NULL,
	CONSTRAINT `build_slots_deployment_id` PRIMARY KEY(`deployment_id`)
);

CREATE INDEX `idx_workspace` ON `build_slots` (`workspace_id`);

