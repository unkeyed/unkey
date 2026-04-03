CREATE TABLE `vertical_autoscaling_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`update_mode` enum('off','initial','recreate','in_place_or_recreate') NOT NULL DEFAULT 'off',
	`controlled_resources` enum('cpu','memory','both') NOT NULL DEFAULT 'both',
	`controlled_values` enum('requests','requests_and_limits') NOT NULL DEFAULT 'requests',
	`cpu_min_millicores` int unsigned,
	`cpu_max_millicores` int unsigned,
	`memory_min_mib` int unsigned,
	`memory_max_mib` int unsigned,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `vertical_autoscaling_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `vertical_autoscaling_policies_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_idx` ON `vertical_autoscaling_policies` (`workspace_id`);

