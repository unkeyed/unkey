CREATE TABLE `apis` (
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`ip_whitelist` varchar(512),
	`auth_type` enum('key','jwt'),
	`key_auth_id` varchar(256),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`delete_protection` boolean DEFAULT false,
	CONSTRAINT `apis_id` PRIMARY KEY(`id`),
	CONSTRAINT `apis_key_auth_id_unique` UNIQUE(`key_auth_id`)
);

CREATE TABLE `keys_permissions` (
	`temp_id` bigint AUTO_INCREMENT NOT NULL,
	`key_id` varchar(256) NOT NULL,
	`permission_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_permissions_key_id_permission_id_workspace_id` PRIMARY KEY(`key_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `keys_permissions_temp_id_unique` UNIQUE(`temp_id`),
	CONSTRAINT `key_id_permission_id_idx` UNIQUE(`key_id`,`permission_id`)
);

CREATE TABLE `keys_roles` (
	`key_id` varchar(256) NOT NULL,
	`role_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_roles_role_id_key_id_workspace_id` PRIMARY KEY(`role_id`,`key_id`,`workspace_id`),
	CONSTRAINT `unique_key_id_role_id` UNIQUE(`key_id`,`role_id`)
);

CREATE TABLE `permissions` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`slug` varchar(128) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `permissions_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_slug_per_workspace_idx` UNIQUE(`workspace_id`,`slug`)
);

CREATE TABLE `roles` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`name`,`workspace_id`)
);

CREATE TABLE `roles_permissions` (
	`role_id` varchar(256) NOT NULL,
	`permission_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_permissions_role_id_permission_id_workspace_id` PRIMARY KEY(`role_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `unique_tuple_permission_id_role_id` UNIQUE(`permission_id`,`role_id`)
);

CREATE TABLE `key_auth` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`store_encrypted_keys` boolean NOT NULL DEFAULT false,
	`default_prefix` varchar(8),
	`default_bytes` int DEFAULT 16,
	`size_approx` int NOT NULL DEFAULT 0,
	`size_last_updated_at` bigint NOT NULL DEFAULT 0,
	CONSTRAINT `key_auth_id` PRIMARY KEY(`id`)
);

CREATE TABLE `encrypted_keys` (
	`workspace_id` varchar(256) NOT NULL,
	`key_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	`encrypted` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	CONSTRAINT `key_id_idx` UNIQUE(`key_id`)
);

CREATE TABLE `keys` (
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
	`refill_amount` int,
	`last_refill_at` datetime(3),
	`enabled` boolean NOT NULL DEFAULT true,
	`remaining_requests` int,
	`ratelimit_async` boolean,
	`ratelimit_limit` int,
	`ratelimit_duration` bigint,
	`environment` varchar(256),
	CONSTRAINT `keys_id` PRIMARY KEY(`id`),
	CONSTRAINT `hash_idx` UNIQUE(`hash`)
);

CREATE TABLE `vercel_bindings` (
	`id` varchar(256) NOT NULL,
	`integration_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`environment` enum('development','preview','production') NOT NULL,
	`resource_id` varchar(256) NOT NULL,
	`resource_type` enum('rootKey','apiId') NOT NULL,
	`vercel_env_id` varchar(256) NOT NULL,
	`last_edited_by` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `vercel_bindings_id` PRIMARY KEY(`id`),
	CONSTRAINT `project_environment_resource_type_idx` UNIQUE(`project_id`,`environment`,`resource_type`)
);

CREATE TABLE `vercel_integrations` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`team_id` varchar(256),
	`access_token` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `vercel_integrations_id` PRIMARY KEY(`id`)
);

CREATE TABLE `ratelimit_namespaces` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_namespaces_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`workspace_id`,`name`)
);

CREATE TABLE `ratelimit_overrides` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`namespace_id` varchar(256) NOT NULL,
	`identifier` varchar(512) NOT NULL,
	`limit` int NOT NULL,
	`duration` int NOT NULL,
	`async` boolean,
	`sharding` enum('edge'),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_overrides_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_identifier_per_namespace_idx` UNIQUE(`namespace_id`,`identifier`)
);

CREATE TABLE `workspaces` (
	`id` varchar(256) NOT NULL,
	`org_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`partition_id` varchar(256),
	`plan` enum('free','pro','enterprise') DEFAULT 'free',
	`tier` varchar(256) DEFAULT 'Free',
	`stripe_customer_id` varchar(256),
	`stripe_subscription_id` varchar(256),
	`beta_features` json NOT NULL,
	`features` json NOT NULL,
	`subscriptions` json,
	`enabled` boolean NOT NULL DEFAULT true,
	`delete_protection` boolean DEFAULT false,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `workspaces_id` PRIMARY KEY(`id`),
	CONSTRAINT `workspaces_org_id_unique` UNIQUE(`org_id`)
);

CREATE TABLE `key_migration_errors` (
	`id` varchar(256) NOT NULL,
	`migration_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`message` json NOT NULL,
	CONSTRAINT `key_migration_errors_id` PRIMARY KEY(`id`)
);

CREATE TABLE `identities` (
	`id` varchar(256) NOT NULL,
	`external_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`environment` varchar(256) NOT NULL DEFAULT 'default',
	`meta` json,
	`deleted` boolean NOT NULL DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `identities_id` PRIMARY KEY(`id`),
	CONSTRAINT `workspace_id_external_id_deleted_idx` UNIQUE(`workspace_id`,`external_id`,`deleted`)
);

CREATE TABLE `ratelimits` (
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`key_id` varchar(256),
	`identity_id` varchar(256),
	`limit` int NOT NULL,
	`duration` bigint NOT NULL,
	`auto_apply` boolean NOT NULL DEFAULT false,
	CONSTRAINT `ratelimits_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_name_per_key_idx` UNIQUE(`key_id`,`name`),
	CONSTRAINT `unique_name_per_identity_idx` UNIQUE(`identity_id`,`name`)
);

CREATE TABLE `quota` (
	`workspace_id` varchar(256) NOT NULL,
	`requests_per_month` bigint NOT NULL DEFAULT 0,
	`logs_retention_days` int NOT NULL DEFAULT 0,
	`audit_logs_retention_days` int NOT NULL DEFAULT 0,
	`team` boolean NOT NULL DEFAULT false,
	CONSTRAINT `quota_workspace_id` PRIMARY KEY(`workspace_id`)
);

CREATE TABLE `audit_log` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	`bucket_id` varchar(256) NOT NULL,
	`event` varchar(256) NOT NULL,
	`time` bigint NOT NULL,
	`display` varchar(256) NOT NULL,
	`remote_ip` varchar(256),
	`user_agent` varchar(256),
	`actor_type` varchar(256) NOT NULL,
	`actor_id` varchar(256) NOT NULL,
	`actor_name` varchar(256),
	`actor_meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_id` PRIMARY KEY(`id`)
);

CREATE TABLE `audit_log_bucket` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`retention_days` int,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`delete_protection` boolean DEFAULT false,
	CONSTRAINT `audit_log_bucket_id` PRIMARY KEY(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`workspace_id`,`name`)
);

CREATE TABLE `audit_log_target` (
	`workspace_id` varchar(256) NOT NULL,
	`bucket_id` varchar(256) NOT NULL,
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	`audit_log_id` varchar(256) NOT NULL,
	`display_name` varchar(256) NOT NULL,
	`type` varchar(256) NOT NULL,
	`id` varchar(256) NOT NULL,
	`name` varchar(256),
	`meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_target_audit_log_id_id_pk` PRIMARY KEY(`audit_log_id`,`id`)
);

CREATE TABLE `partitions` (
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`description` text,
	`aws_account_id` varchar(256) NOT NULL,
	`region` varchar(256) NOT NULL,
	`ip_v4_address` varchar(15),
	`ip_v6_address` varchar(39),
	`status` enum('active','draining','inactive') NOT NULL DEFAULT 'active',
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `partitions_id` PRIMARY KEY(`id`)
);

CREATE TABLE `projects` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`partition_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`git_repository_url` varchar(500),
	`default_branch` varchar(256) DEFAULT 'main',
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `projects_id` PRIMARY KEY(`id`),
	CONSTRAINT `workspace_slug_idx` UNIQUE(`workspace_id`,`slug`)
);

CREATE TABLE `rootfs_images` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`s3_bucket` varchar(256) NOT NULL,
	`s3_key` varchar(500) NOT NULL,
	`size_bytes` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `rootfs_images_id` PRIMARY KEY(`id`)
);

CREATE TABLE `builds` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`deployment_id` varchar(256) NOT NULL,
	`rootfs_image_id` varchar(256),
	`git_commit_sha` varchar(40),
	`git_branch` varchar(256),
	`status` enum('pending','running','succeeded','failed','cancelled') NOT NULL DEFAULT 'pending',
	`build_tool` enum('docker','depot','custom') NOT NULL DEFAULT 'docker',
	`error_message` text,
	`started_at` bigint,
	`completed_at` bigint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `builds_id` PRIMARY KEY(`id`)
);

CREATE TABLE `deployment_steps` (
	`deployment_id` varchar(256) NOT NULL,
	`status` enum('pending','downloading_docker_image','building_rootfs','uploading_rootfs','creating_vm','booting_vm','assigning_domains','completed','failed') NOT NULL,
	`message` text,
	`error_message` text,
	`created_at` bigint NOT NULL,
	CONSTRAINT `deployment_steps_deployment_id_status_pk` PRIMARY KEY(`deployment_id`,`status`)
);

CREATE TABLE `deployments` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`environment` enum('production','preview') NOT NULL DEFAULT 'preview',
	`build_id` varchar(256),
	`rootfs_image_id` varchar(256) NOT NULL,
	`git_commit_sha` varchar(40),
	`git_branch` varchar(256),
	`config_snapshot` json NOT NULL,
	`openapi_spec` text,
	`status` enum('pending','building','deploying','active','failed','archived') NOT NULL DEFAULT 'pending',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployments_id` PRIMARY KEY(`id`)
);

CREATE TABLE `acme_users` (
	`id` bigint unsigned NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`encrypted_key` varchar(255) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_users_id` PRIMARY KEY(`id`)
);

CREATE TABLE `hostname_routes` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`hostname` varchar(256) NOT NULL,
	`deployment_id` varchar(256) NOT NULL,
	`is_enabled` boolean NOT NULL DEFAULT true,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `hostname_routes_id` PRIMARY KEY(`id`),
	CONSTRAINT `hostname_idx` UNIQUE(`hostname`)
);

CREATE TABLE `domain_challenges` (
	`id` bigint unsigned NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`domain_id` varchar(255) NOT NULL,
	`token` varchar(255) NOT NULL,
	`authorization` varchar(255) NOT NULL,
	`status` enum('pending','verified','failed','expired') NOT NULL DEFAULT 'pending',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	`expires_at` bigint unsigned NOT NULL,
	CONSTRAINT `domain_challenges_id` PRIMARY KEY(`id`)
);

CREATE TABLE `domains` (
	`id` varchar(255) NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`project_id` varchar(255) NOT NULL,
	`domain` varchar(255) NOT NULL,
	`type` enum('custom','generated') NOT NULL DEFAULT 'generated',
	`subdomain_config` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `domains_id` PRIMARY KEY(`id`),
	CONSTRAINT `domain_idx` UNIQUE(`domain`)
);

CREATE INDEX `workspace_id_idx` ON `apis` (`workspace_id`);
CREATE INDEX `workspace_id_idx` ON `roles` (`workspace_id`);
CREATE INDEX `key_auth_id_deleted_at_idx` ON `keys` (`key_auth_id`,`deleted_at_m`);
CREATE INDEX `idx_keys_on_for_workspace_id` ON `keys` (`for_workspace_id`);
CREATE INDEX `idx_keys_on_workspace_id` ON `keys` (`workspace_id`);
CREATE INDEX `owner_id_idx` ON `keys` (`owner_id`);
CREATE INDEX `identity_id_idx` ON `keys` (`identity_id`);
CREATE INDEX `deleted_at_idx` ON `keys` (`deleted_at_m`);
CREATE INDEX `name_idx` ON `ratelimits` (`name`);
CREATE INDEX `workspace_id_idx` ON `audit_log` (`workspace_id`);
CREATE INDEX `bucket_id_idx` ON `audit_log` (`bucket_id`);
CREATE INDEX `bucket_idx` ON `audit_log` (`bucket`);
CREATE INDEX `event_idx` ON `audit_log` (`event`);
CREATE INDEX `actor_id_idx` ON `audit_log` (`actor_id`);
CREATE INDEX `time_idx` ON `audit_log` (`time`);
CREATE INDEX `bucket` ON `audit_log_target` (`bucket`);
CREATE INDEX `audit_log_id` ON `audit_log_target` (`audit_log_id`);
CREATE INDEX `id_idx` ON `audit_log_target` (`id`);
CREATE INDEX `status_idx` ON `partitions` (`status`);
CREATE INDEX `workspace_idx` ON `projects` (`workspace_id`);
CREATE INDEX `partition_idx` ON `projects` (`partition_id`);
CREATE INDEX `workspace_idx` ON `rootfs_images` (`workspace_id`);
CREATE INDEX `project_idx` ON `rootfs_images` (`project_id`);
CREATE INDEX `workspace_idx` ON `builds` (`workspace_id`);
CREATE INDEX `project_idx` ON `builds` (`project_id`);
CREATE INDEX `status_idx` ON `builds` (`status`);
CREATE INDEX `rootfs_image_idx` ON `builds` (`rootfs_image_id`);
CREATE INDEX `idx_deployment_id_created_at` ON `deployment_steps` (`deployment_id`,`created_at`);
CREATE INDEX `workspace_idx` ON `deployments` (`workspace_id`);
CREATE INDEX `project_idx` ON `deployments` (`project_id`);
CREATE INDEX `environment_idx` ON `deployments` (`environment`);
CREATE INDEX `status_idx` ON `deployments` (`status`);
CREATE INDEX `rootfs_image_idx` ON `deployments` (`rootfs_image_id`);
CREATE INDEX `domain_idx` ON `acme_users` (`workspace_id`);
CREATE INDEX `workspace_idx` ON `hostname_routes` (`workspace_id`);
CREATE INDEX `project_idx` ON `hostname_routes` (`project_id`);
CREATE INDEX `deployment_idx` ON `hostname_routes` (`deployment_id`);
CREATE INDEX `domainIdWorkspaceId_idx` ON `domain_challenges` (`domain_id`,`workspace_id`);
CREATE INDEX `workspace_idx` ON `domains` (`workspace_id`);
CREATE INDEX `project_idx` ON `domains` (`project_id`);
