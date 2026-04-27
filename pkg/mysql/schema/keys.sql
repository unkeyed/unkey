CREATE TABLE `keys` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`key_auth_id` varchar(256) NOT NULL,
	`hash` varchar(256) NOT NULL,
	`start` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`for_workspace_id` varchar(256),
	`name` varchar(256),
	`owner_id` varchar(256),
	`identity_id` varchar(256),
	`meta` text,
	`expires` datetime(3),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`refill_day` tinyint,
	`refill_amount` bigint unsigned,
	`last_refill_at` datetime(3),
	`enabled` boolean NOT NULL DEFAULT true,
	`remaining_requests` bigint unsigned,
	`environment` varchar(256),
	`last_used_at` bigint unsigned NOT NULL DEFAULT 0,
	`pending_migration_id` varchar(256),
	CONSTRAINT `keys_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_id_unique` UNIQUE(`id`),
	CONSTRAINT `hash_idx` UNIQUE(`hash`)
);

CREATE INDEX `key_auth_id_deleted_at_idx` ON `keys` (`key_auth_id`,`deleted_at_m`,`id`);

CREATE INDEX `idx_keys_on_for_workspace_id` ON `keys` (`for_workspace_id`);

CREATE INDEX `pending_migration_id_idx` ON `keys` (`pending_migration_id`);

CREATE INDEX `idx_keys_on_workspace_id` ON `keys` (`workspace_id`);

CREATE INDEX `owner_id_idx` ON `keys` (`owner_id`);

CREATE INDEX `identity_id_idx` ON `keys` (`identity_id`,`key_auth_id`,`id`);

CREATE INDEX `idx_keys_refill` ON `keys` (`refill_amount`,`deleted_at_m`);

