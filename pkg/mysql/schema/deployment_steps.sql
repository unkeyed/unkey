CREATE TABLE `deployment_steps` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`project_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`environment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`deployment_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`step` enum('queued','starting','building','deploying','network','finalizing') NOT NULL DEFAULT 'queued',
	`started_at` bigint unsigned NOT NULL,
	`ended_at` bigint unsigned,
	`error` varchar(512),
	CONSTRAINT `deployment_steps_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_step_per_deployment` UNIQUE(`deployment_id`,`step`)
);

CREATE INDEX `workspace_idx` ON `deployment_steps` (`workspace_id`);

CREATE INDEX `step_started_at_idx` ON `deployment_steps` (`step`,`started_at`,`ended_at`);

