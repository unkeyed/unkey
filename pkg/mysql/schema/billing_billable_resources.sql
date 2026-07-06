CREATE TABLE `billing_billable_resources` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`resource_type` enum('keyspace','namespace') NOT NULL,
	`resource_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `billing_billable_resources_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `billing_billable_resources_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_resource_idx` UNIQUE(`workspace_id`,`resource_type`,`resource_id`)
);

