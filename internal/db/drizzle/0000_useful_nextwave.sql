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
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `keys_roles` (
	`key_id` varchar(256) NOT NULL,
	`role_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_roles_role_id_key_id_workspace_id` PRIMARY KEY(`role_id`,`key_id`,`workspace_id`),
	CONSTRAINT `unique_key_id_role_id` UNIQUE(`key_id`,`role_id`)
);
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `roles_permissions` (
	`role_id` varchar(256) NOT NULL,
	`permission_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_permissions_role_id_permission_id_workspace_id` PRIMARY KEY(`role_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `unique_tuple_permission_id_role_id` UNIQUE(`permission_id`,`role_id`)
);
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `encrypted_keys` (
	`workspace_id` varchar(256) NOT NULL,
	`key_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	`encrypted` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	CONSTRAINT `key_id_idx` UNIQUE(`key_id`)
);
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `key_migration_errors` (
	`id` varchar(256) NOT NULL,
	`migration_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`message` json NOT NULL,
	CONSTRAINT `key_migration_errors_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `quota` (
	`workspace_id` varchar(256) NOT NULL,
	`requests_per_month` bigint NOT NULL DEFAULT 0,
	`logs_retention_days` int NOT NULL DEFAULT 0,
	`audit_logs_retention_days` int NOT NULL DEFAULT 0,
	`team` boolean NOT NULL DEFAULT false,
	CONSTRAINT `quota_workspace_id` PRIMARY KEY(`workspace_id`)
);
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
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
--> statement-breakpoint
CREATE TABLE `environments` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`description` varchar(255),
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `environments_id` PRIMARY KEY(`id`),
	CONSTRAINT `environments_workspace_id_slug_idx` UNIQUE(`workspace_id`,`slug`)
);
--> statement-breakpoint
CREATE TABLE `projects` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
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
--> statement-breakpoint
CREATE TABLE `deployments` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`environment_id` varchar(256) NOT NULL,
	`git_commit_sha` varchar(40),
	`git_branch` varchar(256),
	`git_commit_message` text,
	`git_commit_author_name` varchar(256),
	`git_commit_author_email` varchar(256),
	`git_commit_author_username` varchar(256),
	`git_commit_author_avatar_url` varchar(512),
	`git_commit_timestamp` bigint,
	`runtime_config` json NOT NULL,
	`openapi_spec` text,
	`status` enum('pending','building','deploying','network','ready','failed') NOT NULL DEFAULT 'pending',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployments_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE TABLE `deployment_steps` (
	`deployment_id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`status` enum('pending','downloading_docker_image','building_rootfs','uploading_rootfs','creating_vm','booting_vm','assigning_domains','completed','failed') NOT NULL,
	`message` varchar(1024) NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `deployment_steps_deployment_id_status_pk` PRIMARY KEY(`deployment_id`,`status`)
);
--> statement-breakpoint
CREATE TABLE `acme_users` (
	`id` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(255) NOT NULL,
	`encrypted_key` text NOT NULL,
	`registration_uri` text,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_users_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE TABLE `domains` (
	`id` varchar(256) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256),
	`deployment_id` varchar(256),
	`domain` varchar(256) NOT NULL,
	`type` enum('custom','wildcard') NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `domains_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE TABLE `acme_challenges` (
	`id` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`domain_id` varchar(256) NOT NULL,
	`token` varchar(256) NOT NULL,
	`type` enum('HTTP-01','DNS-01') NOT NULL,
	`authorization` varchar(256) NOT NULL,
	`status` enum('waiting','pending','verified','failed') NOT NULL,
	`expires_at` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_challenges_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE INDEX `workspace_id_idx` ON `apis` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `workspace_id_idx` ON `roles` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `key_auth_id_deleted_at_idx` ON `keys` (`key_auth_id`,`deleted_at_m`);--> statement-breakpoint
CREATE INDEX `idx_keys_on_for_workspace_id` ON `keys` (`for_workspace_id`);--> statement-breakpoint
CREATE INDEX `idx_keys_on_workspace_id` ON `keys` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `owner_id_idx` ON `keys` (`owner_id`);--> statement-breakpoint
CREATE INDEX `identity_id_idx` ON `keys` (`identity_id`);--> statement-breakpoint
CREATE INDEX `deleted_at_idx` ON `keys` (`deleted_at_m`);--> statement-breakpoint
CREATE INDEX `name_idx` ON `ratelimits` (`name`);--> statement-breakpoint
CREATE INDEX `workspace_id_idx` ON `audit_log` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `bucket_id_idx` ON `audit_log` (`bucket_id`);--> statement-breakpoint
CREATE INDEX `bucket_idx` ON `audit_log` (`bucket`);--> statement-breakpoint
CREATE INDEX `event_idx` ON `audit_log` (`event`);--> statement-breakpoint
CREATE INDEX `actor_id_idx` ON `audit_log` (`actor_id`);--> statement-breakpoint
CREATE INDEX `time_idx` ON `audit_log` (`time`);--> statement-breakpoint
CREATE INDEX `bucket` ON `audit_log_target` (`bucket`);--> statement-breakpoint
CREATE INDEX `audit_log_id` ON `audit_log_target` (`audit_log_id`);--> statement-breakpoint
CREATE INDEX `id_idx` ON `audit_log_target` (`id`);--> statement-breakpoint
CREATE INDEX `workspace_idx` ON `projects` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `workspace_idx` ON `deployments` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `project_idx` ON `deployments` (`project_id`);--> statement-breakpoint
CREATE INDEX `status_idx` ON `deployments` (`status`);--> statement-breakpoint
CREATE INDEX `domain_idx` ON `acme_users` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `workspace_idx` ON `domains` (`workspace_id`);--> statement-breakpoint
CREATE INDEX `project_idx` ON `domains` (`project_id`);--> statement-breakpoint
CREATE INDEX `workspace_idx` ON `acme_challenges` (`workspace_id`);
