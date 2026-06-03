CREATE TABLE `deletions` (
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`resource_type` varchar(64) NOT NULL,
	`resource_id` varchar(256) NOT NULL,
	`delete_permanently_at` bigint NOT NULL,
	CONSTRAINT `deletions_id` PRIMARY KEY(`id`),
	CONSTRAINT `deletions_resource_idx` UNIQUE(`resource_type`,`resource_id`)
);

CREATE INDEX `deletions_workspace_due_idx` ON `deletions` (`workspace_id`,`delete_permanently_at`);

CREATE INDEX `deletions_due_idx` ON `deletions` (`delete_permanently_at`);

