-- apis represents a user-facing API managed by Unkey. Each API belongs to a
-- workspace and can optionally be linked to a key_auth keyspace for key-based
-- authentication.
CREATE TABLE `apis` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- comma separated IPs
	`ip_whitelist` varchar(512),
	`auth_type` enum('key','jwt'),
	-- References key_auth.id
	`key_auth_id` varchar(256),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`delete_protection` boolean DEFAULT false,
	CONSTRAINT `apis_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `apis_id_unique` UNIQUE(`id`),
	CONSTRAINT `apis_key_auth_id_unique` UNIQUE(`key_auth_id`)
);

-- keys_permissions is a join table that assigns individual permissions to keys
-- within a workspace.
CREATE TABLE `keys_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References keys.id
	`key_id` varchar(256) NOT NULL,
	-- References permissions.id
	`permission_id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_permissions_key_id_permission_id_workspace_id` UNIQUE(`key_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `key_id_permission_id_idx` UNIQUE(`key_id`,`permission_id`)
);

-- keys_roles is a join table that assigns roles to keys within a workspace.
CREATE TABLE `keys_roles` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References keys.id
	`key_id` varchar(256) NOT NULL,
	-- References roles.id
	`role_id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `keys_roles_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_roles_role_id_key_id_workspace_id` UNIQUE(`role_id`,`key_id`,`workspace_id`),
	CONSTRAINT `unique_key_id_role_id` UNIQUE(`key_id`,`role_id`)
);

-- permissions defines fine-grained permissions within a workspace. Permissions
-- are assigned to keys directly via keys_permissions or indirectly via roles.
CREATE TABLE `permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`slug` varchar(128) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `permissions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_slug_per_workspace_idx` UNIQUE(`workspace_id`,`slug`)
);

-- roles groups permissions into named bundles within a workspace. Roles are
-- assigned to keys via keys_roles and grant all associated permissions.
CREATE TABLE `roles` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`description` varchar(512),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `roles_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`name`,`workspace_id`)
);

-- roles_permissions is a join table that assigns permissions to roles within a
-- workspace.
CREATE TABLE `roles_permissions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References roles.id
	`role_id` varchar(256) NOT NULL,
	-- References permissions.id
	`permission_id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	CONSTRAINT `roles_permissions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `roles_permissions_role_id_permission_id_workspace_id` UNIQUE(`role_id`,`permission_id`,`workspace_id`),
	CONSTRAINT `unique_tuple_permission_id_role_id` UNIQUE(`permission_id`,`role_id`)
);

-- key_auth represents a keyspace — a collection of API keys sharing common
-- configuration such as prefix, byte length, and encryption settings.
CREATE TABLE `key_auth` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	`store_encrypted_keys` boolean NOT NULL DEFAULT false,
	`default_prefix` varchar(8),
	`default_bytes` int DEFAULT 16,
	-- Approximate key count in this keyspace, accurate at the size_last_updated_at
	-- timestamp. If size_last_updated_at is older than 1 minute, revalidate by
	-- counting all keys and updating this field.
	`size_approx` int NOT NULL DEFAULT 0,
	`size_last_updated_at` bigint NOT NULL DEFAULT 0,
	CONSTRAINT `key_auth_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_auth_id_unique` UNIQUE(`id`)
);

-- encrypted_keys stores the encrypted form of API keys that are configured to be
-- retrievable. Not every key has a row here; only keys where the user opted in
-- to encrypted storage so the plaintext can be recovered later.
CREATE TABLE `encrypted_keys` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References keys.id
	`key_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	`encrypted` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	CONSTRAINT `encrypted_keys_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_id_idx` UNIQUE(`key_id`)
);

-- key_migrations stores how to verify keys that were migrated from an external
-- platform. The algorithm tells us how the original key was hashed so we can
-- verify it. After a migrated key is used once, we re-hash it as a regular key
-- and the migration is complete.
CREATE TABLE `key_migrations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`algorithm` enum('sha256','github.com/seamapi/prefixed-api-key') NOT NULL,
	CONSTRAINT `key_migrations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `key_migrations_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_id_per_workspace_id` UNIQUE(`id`,`workspace_id`)
);

-- keys stores individual API keys. Each key belongs to a key_auth keyspace and
-- a workspace. Keys support optional expiration, usage limits with automatic
-- refills, environment tagging, and migration from external platforms.
CREATE TABLE `keys` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References key_auth.id
	`key_auth_id` varchar(256) NOT NULL,
	`hash` varchar(256) NOT NULL,
	`start` varchar(256) NOT NULL,
	-- References workspaces.id. The workspace that owns the key.
	`workspace_id` varchar(256) NOT NULL,
	-- References workspaces.id. For internal keys, this is the workspace that the
	-- key is for. The owning workspace is an internal one, defined in
	-- env.UNKEY_WORKSPACE_ID. However in order to filter and display the keys in
	-- the UI, we need to know which user/org the key is for. Not used for user keys.
	`for_workspace_id` varchar(256),
	`name` varchar(256),
	`owner_id` varchar(256),
	-- References identities.id
	`identity_id` varchar(256),
	`meta` text,
	`expires` datetime(3),
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	-- Day of the month on which remaining uses are refilled.
	-- 1 = first of the month, 31 = 31st or last available day, null = every day.
	`refill_day` tinyint,
	`refill_amount` int,
	`last_refill_at` datetime(3),
	-- Whether the key is enabled or disabled.
	`enabled` boolean NOT NULL DEFAULT true,
	-- Limit on how many times a key can be verified before it becomes invalid.
	`remaining_requests` int,
	-- Optional environment flag for users to divide keys (e.g. "live" vs "test").
	-- This is a free-form string; schema-level enforcement should happen at the
	-- key_auth level instead.
	`environment` varchar(256),
	`last_used_at` bigint unsigned NOT NULL DEFAULT 0,
	-- References key_migrations.id. If set, determines how to verify this key.
	`pending_migration_id` varchar(256),
	CONSTRAINT `keys_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `keys_id_unique` UNIQUE(`id`),
	CONSTRAINT `hash_idx` UNIQUE(`hash`)
);

-- @deprecated: vercel_bindings is scheduled for deprecation. It maps Vercel
-- project environments to Unkey resources (API IDs, root keys).
CREATE TABLE `vercel_bindings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References vercel_integrations.id
	`integration_id` varchar(256) NOT NULL,
	-- References workspaces.id
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
	CONSTRAINT `vercel_bindings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `vercel_bindings_id_unique` UNIQUE(`id`),
	CONSTRAINT `project_environment_resource_type_idx` UNIQUE(`project_id`,`environment`,`resource_type`)
);

-- @deprecated: vercel_integrations is scheduled for deprecation. It stores
-- workspace-level Vercel integration credentials.
CREATE TABLE `vercel_integrations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`team_id` varchar(256),
	`access_token` varchar(256) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `vercel_integrations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `vercel_integrations_id_unique` UNIQUE(`id`)
);

-- ratelimit_namespaces defines user-created groupings for rate limits (e.g.
-- "api", "webhooks"). Namespaces scope overrides and are used for analytics.
CREATE TABLE `ratelimit_namespaces` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(512) NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_namespaces_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimit_namespaces_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_workspace_idx` UNIQUE(`workspace_id`,`name`)
);

-- ratelimit_overrides allows per-identifier rate limit customization within a
-- namespace. For example, giving a specific user a higher limit than the default.
CREATE TABLE `ratelimit_overrides` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References ratelimit_namespaces.id
	`namespace_id` varchar(256) NOT NULL,
	`identifier` varchar(512) NOT NULL,
	`limit` int NOT NULL,
	-- window duration in milliseconds
	`duration` int NOT NULL,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `ratelimit_overrides_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimit_overrides_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_identifier_per_namespace_idx` UNIQUE(`namespace_id`,`identifier`)
);

-- workspaces is the top-level organizational unit. Each workspace maps to a
-- Clerk organization and contains all resources (APIs, keys, projects, etc.).
CREATE TABLE `workspaces` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`org_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	-- slug is used for the workspace URL
	`slug` varchar(64) NOT NULL,
	`k8s_namespace` varchar(256),
	`tier` varchar(256) DEFAULT 'Free',
	`stripe_customer_id` varchar(256),
	`stripe_subscription_id` varchar(256),
	-- Feature flags. betaFeatures may be toggled by the user for early access.
	`beta_features` json NOT NULL,
	-- @deprecated: most customers are on stripe subscriptions instead
	`subscriptions` json,
	-- If the workspace is disabled, all API requests will be rejected.
	`enabled` boolean NOT NULL DEFAULT true,
	`delete_protection` boolean DEFAULT false,
	`created_at_m` bigint NOT NULL DEFAULT 0,
	`updated_at_m` bigint,
	`deleted_at_m` bigint,
	CONSTRAINT `workspaces_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspaces_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspaces_org_id_unique` UNIQUE(`org_id`),
	CONSTRAINT `workspaces_slug_unique` UNIQUE(`slug`),
	CONSTRAINT `workspaces_k8s_namespace_unique` UNIQUE(`k8s_namespace`)
);

-- The external_id creates a reference to the user's existing data. They likely
-- have an organization or user ID at hand.
CREATE TABLE `identities` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	`external_id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`environment` varchar(256) NOT NULL DEFAULT 'default',
	`meta` json,
	`deleted` boolean NOT NULL DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `identities_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `identities_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_id_external_id_deleted_idx` UNIQUE(`workspace_id`,`external_id`,`deleted`)
);

-- Ratelimits can be attached to a key or identity and are referenced by name
-- when verifying a key.
CREATE TABLE `ratelimits` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- The name is used to reference this limit when verifying a key.
	`name` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	-- References keys.id. Either key_id or identity_id may be defined, not both.
	`key_id` varchar(256),
	-- References identities.id. Either key_id or identity_id may be defined, not both.
	`identity_id` varchar(256),
	`limit` int NOT NULL,
	-- milliseconds
	`duration` bigint NOT NULL,
	-- If enabled, this limit is applied when verifying a key whether or not the
	-- caller specified the name in the request.
	`auto_apply` boolean NOT NULL DEFAULT false,
	CONSTRAINT `ratelimits_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `ratelimits_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_name_per_key_idx` UNIQUE(`key_id`,`name`),
	CONSTRAINT `unique_name_per_identity_idx` UNIQUE(`identity_id`,`name`)
);

-- quotas represents the resource allocation and retention limits for workspaces.
-- Each workspace has a single quota record that defines its operational boundaries,
-- including the maximum number of requests allowed per month and data retention
-- policies. These settings control service availability and billing.
CREATE TABLE `quota` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- Maximum billable API requests per calendar month. Exceeding this limit may
	-- result in throttling or nudges to upgrade. Default 0 = no requests allowed
	-- without explicit configuration.
	`requests_per_month` bigint NOT NULL DEFAULT 0,
	-- How many days operational logs are stored before automatic deletion.
	`logs_retention_days` int NOT NULL DEFAULT 0,
	-- How many days audit logs are stored before automatic deletion. Audit logs
	-- contain security-relevant events and may have different retention
	-- requirements than operational logs.
	`audit_logs_retention_days` int NOT NULL DEFAULT 0,
	-- Whether team collaboration features are enabled. When true, the workspace
	-- supports multiple users with different roles and permissions.
	`team` boolean NOT NULL DEFAULT false,
	-- Maximum API requests allowed per duration window. NULL = unlimited.
	`ratelimit_api_limit` int unsigned,
	-- Time window in milliseconds for the workspace API rate limit. Used together
	-- with ratelimit_api_limit. NULL = unlimited.
	`ratelimit_api_duration` int unsigned,
	-- Total CPU resources (in millicores, where 1000 = 1 CPU) a workspace may
	-- allocate at the same time. New deployments exceeding this limit are rejected.
	`allocated_cpu_millicores_total` int unsigned NOT NULL DEFAULT 10000,
	-- Total memory resources (in MiB) a workspace may allocate at the same time.
	-- New deployments exceeding this limit are rejected.
	`allocated_memory_mib_total` int unsigned NOT NULL DEFAULT 20480,
	CONSTRAINT `quota_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `quota_workspace_id_unique` UNIQUE(`workspace_id`)
);

-- audit_log records security-relevant events within a workspace such as key
-- creation, deletion, and permission changes.
CREATE TABLE `audit_log` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(256) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- The name of the bucket that the audit log belongs to.
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	-- @deprecated
	`bucket_id` varchar(256) NOT NULL,
	`event` varchar(256) NOT NULL,
	-- When the event happened
	`time` bigint NOT NULL,
	-- A human readable description of the event
	`display` varchar(256) NOT NULL,
	`remote_ip` varchar(256),
	`user_agent` varchar(256),
	`actor_type` varchar(256) NOT NULL,
	`actor_id` varchar(256) NOT NULL,
	`actor_name` varchar(256),
	`actor_meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `audit_log_id_unique` UNIQUE(`id`)
);

-- audit_log_target records the resources affected by an audit log event. Each
-- audit log entry can have multiple targets (e.g. a key and the API it belongs to).
CREATE TABLE `audit_log_target` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- @deprecated
	`bucket_id` varchar(256) NOT NULL,
	-- The name of the bucket that the target belongs to.
	`bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
	-- References audit_log.id
	`audit_log_id` varchar(256) NOT NULL,
	-- A human readable name to display in the UI
	`display_name` varchar(256) NOT NULL,
	-- The type of the target
	`type` varchar(256) NOT NULL,
	-- The ID of the target
	`id` varchar(256) NOT NULL,
	-- The name of the target
	`name` varchar(256),
	-- The metadata of the target
	`meta` json,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `audit_log_target_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_id_per_log` UNIQUE(`audit_log_id`,`id`)
);

-- environments represents deployment environments (e.g. production, preview)
-- within an app. Each environment has its own build settings, runtime config,
-- and environment variables.
CREATE TABLE `environments` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References projects.id
	`project_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- URL-safe identifier within workspace
	`slug` varchar(256) NOT NULL,
	`description` varchar(255) NOT NULL DEFAULT '',
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `environments_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `environments_id_unique` UNIQUE(`id`),
	CONSTRAINT `environments_app_slug_idx` UNIQUE(`app_id`,`slug`)
);

-- ClickHouse configuration for workspaces with analytics enabled. Each workspace
-- gets a dedicated user with resource quotas to prevent abuse.
CREATE TABLE `clickhouse_workspace_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- Authentication
	`username` varchar(256) NOT NULL,
	`password_encrypted` text NOT NULL,
	-- Quota window configuration
	`quota_duration_seconds` int NOT NULL DEFAULT 3600,
	`max_queries_per_window` int NOT NULL DEFAULT 1000,
	`max_execution_time_per_window` int NOT NULL DEFAULT 1800,
	-- Per-query limits (prevent cluster crashes)
	`max_query_execution_time` int NOT NULL DEFAULT 30,
	`max_query_memory_bytes` bigint NOT NULL DEFAULT 1000000000,
	`max_query_result_rows` int NOT NULL DEFAULT 10000,
	`created_at` bigint NOT NULL DEFAULT 0,
	`updated_at` bigint,
	CONSTRAINT `clickhouse_workspace_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `clickhouse_workspace_settings_workspace_id_unique` UNIQUE(`workspace_id`),
	CONSTRAINT `clickhouse_workspace_settings_username_unique` UNIQUE(`username`)
);

-- projects is the top-level container for the deployment platform. A project
-- belongs to a workspace and contains one or more apps.
CREATE TABLE `projects` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`name` varchar(256) NOT NULL,
	-- URL-safe identifier within workspace
	`slug` varchar(256) NOT NULL,
	`depot_project_id` varchar(255),
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `projects_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `projects_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_slug_idx` UNIQUE(`workspace_id`,`slug`)
);

-- apps represents a deployable application within a project. Each app has its
-- own build/runtime settings, environments, and deployment history.
CREATE TABLE `apps` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References projects.id
	`project_id` varchar(64) NOT NULL,
	`name` varchar(256) NOT NULL,
	`slug` varchar(256) NOT NULL,
	`default_branch` varchar(256) NOT NULL DEFAULT 'main',
	-- References deployments.id
	`current_deployment_id` varchar(256),
	-- When true, new deployments are blocked from using the production domain.
	-- Requires manual promotion to exit this state, preventing a new push from
	-- accidentally redeploying a buggy version after a rollback.
	`is_rolled_back` boolean NOT NULL DEFAULT false,
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `apps_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `apps_id_unique` UNIQUE(`id`),
	CONSTRAINT `apps_project_slug_idx` UNIQUE(`project_id`,`slug`)
);

-- app_build_settings stores the Docker build configuration for an app in a
-- specific environment. One row per app+environment combination.
CREATE TABLE `app_build_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	`dockerfile` varchar(500) NOT NULL DEFAULT 'Dockerfile',
	`docker_context` varchar(500) NOT NULL DEFAULT '.',
	`watch_paths` json NOT NULL DEFAULT ('[]'),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_build_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_build_settings_app_env_idx` UNIQUE(`app_id`,`environment_id`)
);

-- app_runtime_settings stores the runtime configuration for an app in a specific
-- environment, including port, resource limits, healthchecks, and sentinel config.
CREATE TABLE `app_runtime_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	`port` int NOT NULL DEFAULT 8080,
	-- CPU allocation in millicores (1000 millicores = 1 CPU).
	`cpu_millicores` int NOT NULL DEFAULT 250,
	`memory_mib` int NOT NULL DEFAULT 256,
	`command` json NOT NULL DEFAULT ('[]'),
	-- null = no healthcheck configured
	`healthcheck` json,
	`shutdown_signal` enum('SIGTERM','SIGINT','SIGQUIT','SIGKILL') NOT NULL DEFAULT 'SIGTERM',
	`sentinel_config` longblob NOT NULL,
	-- null = scraping disabled; non-null path (e.g. /openapi.yaml) enables scraping
	`openapi_spec_path` varchar(512),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_runtime_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_runtime_settings_app_env_idx` UNIQUE(`app_id`,`environment_id`)
);

-- One row per region an app+environment should be deployed to. Replaces the
-- region_config JSON column on app_runtime_settings. Presence of a row means
-- "deploy this app/env to this region".
CREATE TABLE `app_regional_settings` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	-- References regions.id
	`region_id` varchar(64) NOT NULL,
	`replicas` int NOT NULL DEFAULT 1,
	-- References horizontal_autoscaling_policies.id. null = no autoscaling.
	`horizontal_autoscaling_policy_id` varchar(64),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_regional_settings_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_app_env_region` UNIQUE(`app_id`,`environment_id`,`region_id`)
);

-- app_environment_variables stores encrypted environment variables for an app in
-- a specific environment. Values are always encrypted via vault.
CREATE TABLE `app_environment_variables` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	`key` varchar(256) NOT NULL,
	-- Always encrypted via vault (contains keyId, nonce, ciphertext in the blob).
	`value` varchar(4096) NOT NULL,
	-- Both types are encrypted in the database:
	--   recoverable: can be decrypted and shown in the UI
	--   writeonly: cannot be read back after creation
	`type` enum('recoverable','writeonly') NOT NULL,
	`description` varchar(255),
	`delete_protection` boolean DEFAULT false,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_environment_variables_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_environment_variables_id_unique` UNIQUE(`id`),
	CONSTRAINT `app_env_id_key` UNIQUE(`app_id`,`environment_id`,`key`)
);

-- deployments represents a single deployment of an app to an environment. Stores
-- a snapshot of all configuration (env vars, command, resources) at deploy time
-- so deployments are reproducible regardless of later settings changes.
CREATE TABLE `deployments` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`k8s_name` varchar(255) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References projects.id
	`project_id` varchar(256) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- The docker image. null until the build is done.
	`image` varchar(256),
	`build_id` varchar(128),
	-- Git information
	`git_commit_sha` varchar(40),
	`git_branch` varchar(256),
	`git_commit_message` text,
	`git_commit_author_handle` varchar(256),
	`git_commit_author_avatar_url` varchar(512),
	-- Unix epoch milliseconds
	`git_commit_timestamp` bigint,
	`sentinel_config` longblob NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`desired_state` enum('running','standby','archived') NOT NULL DEFAULT 'running',
	-- Environment variables snapshot (protobuf: ctrl.v1.SecretsBlob).
	-- Encrypted values from environment_variables at deploy time.
	`encrypted_environment_variables` longblob NOT NULL,
	-- Container command override (e.g. ["./app", "serve"]).
	-- If empty, the container's default entrypoint/cmd is used.
	`command` json NOT NULL DEFAULT ('[]'),
	-- Port the container listens on.
	`port` int NOT NULL DEFAULT 8080,
	`shutdown_signal` enum('SIGTERM','SIGINT','SIGQUIT','SIGKILL') NOT NULL DEFAULT 'SIGTERM',
	-- HTTP healthcheck configuration (null = no healthcheck).
	`healthcheck` json,
	-- PR number (for fork PRs, used to build refs/pull/N/head for BuildKit).
	`pr_number` bigint,
	-- Fork repository full name (e.g. "contributor/repo") for linking to the fork.
	`fork_repository_full_name` varchar(256),
	-- GitHub Deployment ID for status reporting.
	`github_deployment_id` bigint,
	`status` enum('pending','starting','building','deploying','network','finalizing','ready','failed','skipped','awaiting_approval','stopped') NOT NULL DEFAULT 'pending',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployments_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `deployments_id_unique` UNIQUE(`id`),
	CONSTRAINT `deployments_k8s_name_unique` UNIQUE(`k8s_name`),
	CONSTRAINT `deployments_build_id_unique` UNIQUE(`build_id`)
);

-- openapi_specs stores OpenAPI specification documents scraped from deployments
-- or uploaded for portal configurations (similar to Stripe's checkout portal).
CREATE TABLE `openapi_specs` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(128),
	`portal_config_id` varchar(256),
	`content` longblob NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `openapi_specs_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `openapi_specs_id_unique` UNIQUE(`id`),
	CONSTRAINT `workspace_deployment_idx` UNIQUE(`workspace_id`,`deployment_id`),
	CONSTRAINT `workspace_portal_config_idx` UNIQUE(`workspace_id`,`portal_config_id`)
);

-- deployment_steps logs each phase of a deployment with start/end timestamps,
-- so users can see how long each step took (e.g. "building took 30s").
CREATE TABLE `deployment_steps` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(128) NOT NULL,
	-- References projects.id
	`project_id` varchar(128) NOT NULL,
	-- References environments.id
	`environment_id` varchar(128) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(128) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	`step` enum('queued','starting','building','deploying','network','finalizing') NOT NULL DEFAULT 'queued',
	`started_at` bigint unsigned NOT NULL,
	`ended_at` bigint unsigned,
	`error` varchar(512),
	CONSTRAINT `deployment_steps_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_step_per_deployment` UNIQUE(`deployment_id`,`step`)
);

-- deployment_topology maps a deployment to the regions it runs in, along with
-- autoscaling configuration snapshotted at deploy time. Krane use the
-- version column for state synchronization.
CREATE TABLE `deployment_topology` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(64) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(64) NOT NULL,
	-- References regions.id
	`region_id` varchar(64) NOT NULL,
	-- HPA scaling configuration, snapshotted from the autoscaling policy at deploy time.
	-- Minimum number of pod replicas the HPA will maintain.
	`autoscaling_replicas_min` int unsigned NOT NULL DEFAULT 1,
	-- Maximum number of pod replicas the HPA can scale to.
	`autoscaling_replicas_max` int unsigned NOT NULL DEFAULT 1,
	-- Average CPU utilization percentage (0-100) that triggers scale-up. null = use default (80%).
	`autoscaling_threshold_cpu` tinyint unsigned,
	-- Average memory utilization percentage (0-100) that triggers scale-up. null = not used as a signal.
	`autoscaling_threshold_memory` tinyint unsigned,
	-- Version for state synchronization with krane. Updated via Restate
	-- VersioningService on each mutation. Krane track their last-seen
	-- version and request changes after it. Unique per region_id (composite
	-- index with region_id).
	`version` bigint unsigned NOT NULL,
	`desired_status` enum('stopped','running') NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployment_topology_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `unique_region_per_deployment` UNIQUE(`deployment_id`,`region_id`),
	CONSTRAINT `unique_version_per_region` UNIQUE(`region_id`,`version`)
);

-- acme_users stores ACME account credentials per workspace for automated TLS
-- certificate provisioning.
CREATE TABLE `acme_users` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	`encrypted_key` text NOT NULL,
	`registration_uri` text,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_users_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `acme_users_id_unique` UNIQUE(`id`)
);

-- custom_domains tracks user-provided domains and their verification state.
-- Ownership is proven via TXT record, routing is enabled via CNAME verification.
CREATE TABLE `custom_domains` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	-- References projects.id
	`project_id` varchar(256) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(256) NOT NULL,
	`domain` varchar(256) NOT NULL,
	`challenge_type` enum('HTTP-01','DNS-01') NOT NULL,
	`verification_status` enum('pending','verifying','verified','failed') NOT NULL DEFAULT 'pending',
	-- TXT record verification token (e.g. "abc123xyz..."). User adds TXT record:
	-- _unkey.domain.com -> unkey-domain-verify=<token>
	`verification_token` varchar(64) NOT NULL,
	-- Whether the TXT record has been verified (proves ownership).
	`ownership_verified` boolean NOT NULL DEFAULT false,
	-- Whether the CNAME record has been verified (enables routing).
	`cname_verified` boolean NOT NULL DEFAULT false,
	-- Unique CNAME target for this domain (e.g. "k3n5p8x2"). Combined with base
	-- domain to form full target like "k3n5p8x2.cname.unkey.local".
	`target_cname` varchar(256) NOT NULL,
	`last_checked_at` bigint,
	`check_attempts` int NOT NULL DEFAULT 0,
	`verification_error` varchar(512),
	`invocation_id` varchar(256),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `custom_domains_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `custom_domains_id_unique` UNIQUE(`id`),
	CONSTRAINT `custom_domains_target_cname_unique` UNIQUE(`target_cname`),
	CONSTRAINT `unique_domain_workspace_idx` UNIQUE(`workspace_id`,`domain`)
);

-- acme_challenges stores in-flight ACME domain validation challenges. Rows
-- exist until ownership is proven or the challenge expires.
CREATE TABLE `acme_challenges` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References custom_domains.id
	`domain_id` varchar(255) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	`token` varchar(255) NOT NULL,
	`challenge_type` enum('HTTP-01','DNS-01') NOT NULL,
	`authorization` varchar(255) NOT NULL,
	`status` enum('waiting','pending','verified','failed') NOT NULL,
	`expires_at` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `acme_challenges_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `acme_challenges_domain_id_unique` UNIQUE(`domain_id`)
);

-- One row per logical sentinel. Each set of sentinel pods in a single region is
-- one row. Therefore each sentinel also has a single kubernetes service name.
CREATE TABLE `sentinels` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	-- References projects.id
	`project_id` varchar(255) NOT NULL,
	-- References environments.id
	`environment_id` varchar(255) NOT NULL,
	`k8s_name` varchar(64) NOT NULL,
	`k8s_address` varchar(255) NOT NULL,
	-- References regions.id
	`region_id` varchar(255) NOT NULL,
	`image` varchar(255) NOT NULL,
	`desired_state` enum('running','standby','archived') NOT NULL DEFAULT 'running',
	`health` enum('unknown','paused','healthy','unhealthy') NOT NULL DEFAULT 'unknown',
	`desired_replicas` int NOT NULL,
	`available_replicas` int NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	-- Version for state synchronization with krane. Updated via Restate
	-- VersioningService on each mutation. Krane track their last-seen
	-- version and request changes after it. Unique per region (composite index
	-- with region).
	`version` bigint unsigned NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `sentinels_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `sentinels_id_unique` UNIQUE(`id`),
	CONSTRAINT `sentinels_k8s_name_unique` UNIQUE(`k8s_name`),
	CONSTRAINT `sentinels_k8s_address_unique` UNIQUE(`k8s_address`),
	CONSTRAINT `one_env_per_region` UNIQUE(`environment_id`,`region_id`),
	CONSTRAINT `unique_version_per_region` UNIQUE(`region_id`,`version`)
);

-- instances represents individual running pods for a deployment in a region.
-- Updated from kubernetes watch events.
CREATE TABLE `instances` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(255) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	-- References projects.id
	`project_id` varchar(255) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References regions.id
	`region_id` varchar(64) NOT NULL,
	-- Used to apply updates from the kubernetes watch events.
	`k8s_name` varchar(255) NOT NULL,
	-- The kubernetes pod DNS address from the stateful set.
	`address` varchar(255) NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`status` enum('inactive','pending','running','failed') NOT NULL,
	CONSTRAINT `instances_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `instances_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_address_per_region` UNIQUE(`address`,`region_id`),
	CONSTRAINT `unique_k8s_name_per_region` UNIQUE(`k8s_name`,`region_id`)
);

-- certificates stores TLS certificates and their encrypted private keys. Used
-- for custom domains and the *.unkey.app wildcard certificate, managed via ACME.
CREATE TABLE `certificates` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	`hostname` varchar(255) NOT NULL,
	`certificate` text NOT NULL,
	`encrypted_private_key` text NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `certificates_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `certificates_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_hostname` UNIQUE(`hostname`)
);

-- frontline_routes maps fully qualified domain names to deployments. The sticky
-- column controls whether a FQDN follows new deployments or stays pinned.
CREATE TABLE `frontline_routes` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	-- References projects.id
	`project_id` varchar(255) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(255) NOT NULL,
	-- References environments.id
	`environment_id` varchar(255) NOT NULL,
	`fully_qualified_domain_name` varchar(256) NOT NULL,
	-- sticky determines whether a FQDN should get reassigned to the latest deployment:
	--   none: always reassigned to the latest deployment
	--   branch: sticky to the branch (reassigned only when a new deployment on the same branch is created)
	--   environment: sticky to the environment
	--   live: the production route, reassigned on explicit promotion
	--   deployment: permanently pinned to this specific deployment
	`sticky` enum('none','branch','environment','live','deployment') NOT NULL DEFAULT 'none',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `frontline_routes_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `frontline_routes_id_unique` UNIQUE(`id`),
	CONSTRAINT `frontline_routes_fully_qualified_domain_name_unique` UNIQUE(`fully_qualified_domain_name`)
);

-- github_app_installations links a workspace to a GitHub App installation,
-- enabling GitHub-based triggers for deployments.
CREATE TABLE `github_app_installations` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`installation_id` bigint NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `github_app_installations_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `workspace_installation_idx` UNIQUE(`workspace_id`,`installation_id`)
);

-- github_repo_connections links a specific app to a GitHub repository for
-- automatic deployments on push.
CREATE TABLE `github_repo_connections` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	-- References projects.id
	`project_id` varchar(64) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References github_app_installations.installation_id
	`installation_id` bigint NOT NULL,
	`repository_id` bigint NOT NULL,
	`repository_full_name` varchar(500) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `github_repo_connections_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `github_repo_connections_app_id_unique` UNIQUE(`app_id`)
);

-- cilium_network_policies stores Cilium network policy definitions for
-- deployments. Krane synchronize policies using the version column.
CREATE TABLE `cilium_network_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(255) NOT NULL,
	-- References projects.id
	`project_id` varchar(255) NOT NULL,
	-- References apps.id
	`app_id` varchar(64) NOT NULL,
	-- References environments.id
	`environment_id` varchar(255) NOT NULL,
	-- References deployments.id
	`deployment_id` varchar(128) NOT NULL,
	`k8s_name` varchar(64) NOT NULL,
	`k8s_namespace` varchar(255) NOT NULL,
	-- References regions.id
	`region_id` varchar(64) NOT NULL,
	-- JSON representation of the policy.
	`policy` json NOT NULL,
	-- Version for state synchronization with krane. Updated via Restate
	-- VersioningService on each mutation. Krane track their last-seen
	-- version and request changes after it. Unique per region (composite index
	-- with region).
	`version` bigint unsigned NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `cilium_network_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `cilium_network_policies_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_version_per_region` UNIQUE(`region_id`,`version`)
);

-- Tracks kubernetes clusters. Each krane instance heartbeats against the control
-- plane, which writes to this table. May be used as service discovery later to
-- push updates to clusters to speed up reconciliation.
CREATE TABLE `clusters` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References regions.id
	`region_id` varchar(64) NOT NULL,
	`last_heartbeat_at` bigint unsigned NOT NULL,
	CONSTRAINT `clusters_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `clusters_id_unique` UNIQUE(`id`),
	CONSTRAINT `clusters_region_id_unique` UNIQUE(`region_id`)
);

-- regions defines the available deployment regions across cloud providers.
CREATE TABLE `regions` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- e.g. us-east-1, us-west-2
	`name` varchar(64) NOT NULL,
	-- e.g. aws, gcp, azure, local
	`platform` varchar(64) NOT NULL,
	-- Whether this region is available for users to schedule deployments to.
	-- Defaults to true; set to false to hide a region from scheduling.
	`can_schedule` boolean NOT NULL DEFAULT true,
	CONSTRAINT `regions_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `regions_id_unique` UNIQUE(`id`),
	CONSTRAINT `unique_region_per_platform` UNIQUE(`name`,`platform`)
);

-- A reusable horizontal autoscaling policy. Other tables (e.g.
-- app_regional_settings, sentinels) reference this via
-- horizontal_autoscaling_policy_id. If no policy is referenced, static replica
-- counts are used.
CREATE TABLE `horizontal_autoscaling_policies` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	-- References workspaces.id
	`workspace_id` varchar(256) NOT NULL,
	`replicas_min` int NOT NULL,
	`replicas_max` int NOT NULL,
	-- 0-100 percentage thresholds that trigger scaling. null = not used as a signal.
	`memory_threshold` tinyint,
	`cpu_threshold` tinyint,
	`rps_threshold` tinyint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `horizontal_autoscaling_policies_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `horizontal_autoscaling_policies_id_unique` UNIQUE(`id`)
);

CREATE INDEX `workspace_id_idx` ON `apis` (`workspace_id`);
CREATE INDEX `workspace_id_idx` ON `roles` (`workspace_id`);
CREATE INDEX `key_auth_id_deleted_at_idx` ON `keys` (`key_auth_id`,`deleted_at_m`,`id`);
CREATE INDEX `idx_keys_on_for_workspace_id` ON `keys` (`for_workspace_id`);
CREATE INDEX `pending_migration_id_idx` ON `keys` (`pending_migration_id`);
CREATE INDEX `idx_keys_on_workspace_id` ON `keys` (`workspace_id`);
CREATE INDEX `owner_id_idx` ON `keys` (`owner_id`);
CREATE INDEX `identity_id_idx` ON `keys` (`identity_id`,`key_auth_id`,`id`);
CREATE INDEX `idx_keys_refill` ON `keys` (`refill_amount`,`deleted_at_m`);
CREATE INDEX `workspace_id_idx` ON `audit_log` (`workspace_id`);
CREATE INDEX `bucket_id_idx` ON `audit_log` (`bucket_id`);
CREATE INDEX `bucket_idx` ON `audit_log` (`bucket`);
CREATE INDEX `event_idx` ON `audit_log` (`event`);
CREATE INDEX `actor_id_idx` ON `audit_log` (`actor_id`);
CREATE INDEX `time_idx` ON `audit_log` (`time`);
CREATE INDEX `bucket` ON `audit_log_target` (`bucket`);
CREATE INDEX `id_idx` ON `audit_log_target` (`id`);
CREATE INDEX `environments_project_idx` ON `environments` (`project_id`);
CREATE INDEX `apps_workspace_idx` ON `apps` (`workspace_id`);
CREATE INDEX `workspace_idx` ON `app_regional_settings` (`workspace_id`);
CREATE INDEX `workspace_idx` ON `deployments` (`workspace_id`);
CREATE INDEX `project_idx` ON `deployments` (`project_id`);
CREATE INDEX `status_idx` ON `deployments` (`status`);
CREATE INDEX `workspace_idx` ON `deployment_steps` (`workspace_id`);
CREATE INDEX `workspace_idx` ON `deployment_topology` (`workspace_id`);
CREATE INDEX `status_idx` ON `deployment_topology` (`desired_status`);
CREATE INDEX `domain_idx` ON `acme_users` (`workspace_id`);
CREATE INDEX `project_idx` ON `custom_domains` (`project_id`);
CREATE INDEX `verification_status_idx` ON `custom_domains` (`verification_status`);
CREATE INDEX `workspace_idx` ON `acme_challenges` (`workspace_id`);
CREATE INDEX `status_idx` ON `acme_challenges` (`status`);
CREATE INDEX `idx_environment_health_region_routing` ON `sentinels` (`environment_id`,`region_id`,`health`);
CREATE INDEX `idx_deployment_id` ON `instances` (`deployment_id`);
CREATE INDEX `idx_region` ON `instances` (`region_id`);
CREATE INDEX `environment_id_idx` ON `frontline_routes` (`environment_id`);
CREATE INDEX `deployment_id_idx` ON `frontline_routes` (`deployment_id`);
CREATE INDEX `fqdn_environment_deployment_idx` ON `frontline_routes` (`fully_qualified_domain_name`,`environment_id`,`deployment_id`);
CREATE INDEX `installation_id_idx` ON `github_repo_connections` (`installation_id`);
CREATE INDEX `idx_deployment_region` ON `cilium_network_policies` (`deployment_id`,`region_id`);
CREATE INDEX `workspace_idx` ON `horizontal_autoscaling_policies` (`workspace_id`);
