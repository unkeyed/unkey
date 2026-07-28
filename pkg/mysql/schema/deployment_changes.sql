CREATE TABLE `deployment_changes` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`resource_type` enum('deployment_topology','sentinel','cilium_network_policy') NOT NULL,
	`resource_id` varchar(64) NOT NULL,
	`region_id` varchar(64) NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `deployment_changes_pk` PRIMARY KEY(`pk`)
);

CREATE INDEX `idx_region_type_pk` ON `deployment_changes` (`region_id`,`resource_type`,`pk`);

CREATE INDEX `idx_created_at` ON `deployment_changes` (`created_at`);

CREATE INDEX `idx_region_pk` ON `deployment_changes` (`region_id`,`pk`);

