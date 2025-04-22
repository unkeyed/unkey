CREATE TABLE `apis` (
  `id` varchar(256) NOT NULL,
  `name` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `ip_whitelist` varchar(512) DEFAULT NULL,
  `auth_type` enum('key','jwt') DEFAULT NULL,
  `key_auth_id` varchar(256) DEFAULT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  `delete_protection` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `apis_key_auth_id_unique` (`key_auth_id`),
  KEY `workspace_id_idx` (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `audit_log` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
  `bucket_id` varchar(256) NOT NULL,
  `event` varchar(256) NOT NULL,
  `time` bigint NOT NULL,
  `display` varchar(256) NOT NULL,
  `remote_ip` varchar(256) DEFAULT NULL,
  `user_agent` varchar(256) DEFAULT NULL,
  `actor_type` varchar(256) NOT NULL,
  `actor_id` varchar(256) NOT NULL,
  `actor_name` varchar(256) DEFAULT NULL,
  `actor_meta` json DEFAULT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `workspace_id_idx` (`workspace_id`),
  KEY `bucket_id_idx` (`bucket_id`),
  KEY `bucket_idx` (`bucket`),
  KEY `event_idx` (`event`),
  KEY `actor_id_idx` (`actor_id`),
  KEY `time_idx` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `audit_log_bucket` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `name` varchar(256) NOT NULL,
  `retention_days` int DEFAULT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint DEFAULT NULL,
  `delete_protection` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_name_per_workspace_idx` (`workspace_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `audit_log_target` (
  `workspace_id` varchar(256) NOT NULL,
  `bucket_id` varchar(256) NOT NULL,
  `bucket` varchar(256) NOT NULL DEFAULT 'unkey_mutations',
  `audit_log_id` varchar(256) NOT NULL,
  `display_name` varchar(256) NOT NULL,
  `type` varchar(256) NOT NULL,
  `id` varchar(256) NOT NULL,
  `name` varchar(256) DEFAULT NULL,
  `meta` json DEFAULT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`audit_log_id`,`id`),
  KEY `bucket` (`bucket`),
  KEY `audit_log_id` (`audit_log_id`),
  KEY `id_idx` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `encrypted_keys` (
  `workspace_id` varchar(256) NOT NULL,
  `key_id` varchar(256) NOT NULL,
  `created_at` bigint NOT NULL DEFAULT '0',
  `updated_at` bigint DEFAULT NULL,
  `encrypted` varchar(1024) NOT NULL,
  `encryption_key_id` varchar(256) NOT NULL,
  UNIQUE KEY `key_id_idx` (`key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `identities` (
  `id` varchar(256) NOT NULL,
  `external_id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `environment` varchar(256) NOT NULL DEFAULT 'default',
  `created_at` bigint NOT NULL,
  `updated_at` bigint DEFAULT NULL,
  `meta` json DEFAULT NULL,
  `deleted` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `workspace_id_external_id_deleted_idx` (`workspace_id`,`external_id`,`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `key_auth` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  `store_encrypted_keys` tinyint(1) NOT NULL DEFAULT '0',
  `default_prefix` varchar(8) DEFAULT NULL,
  `default_bytes` int DEFAULT '16',
  `size_approx` int NOT NULL DEFAULT '0',
  `size_last_updated_at` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `key_migration_errors` (
  `id` varchar(256) NOT NULL,
  `migration_id` varchar(256) NOT NULL,
  `created_at` bigint NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `message` json NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `keys` (
  `id` varchar(256) NOT NULL,
  `key_auth_id` varchar(256) NOT NULL,
  `hash` varchar(256) NOT NULL,
  `start` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `for_workspace_id` varchar(256) DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `owner_id` varchar(256) DEFAULT NULL,
  `identity_id` varchar(256) DEFAULT NULL,
  `meta` text,
  `expires` datetime(3) DEFAULT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  `refill_day` tinyint DEFAULT NULL,
  `refill_amount` int DEFAULT NULL,
  `last_refill_at` datetime(3) DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `remaining_requests` int DEFAULT NULL,
  `ratelimit_async` tinyint(1) DEFAULT NULL,
  `ratelimit_limit` int DEFAULT NULL,
  `ratelimit_duration` bigint DEFAULT NULL,
  `environment` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `hash_idx` (`hash`),
  KEY `key_auth_id_deleted_at_idx` (`key_auth_id`,`deleted_at_m`),
  KEY `idx_keys_on_for_workspace_id` (`for_workspace_id`),
  KEY `owner_id_idx` (`owner_id`),
  KEY `identity_id_idx` (`identity_id`),
  KEY `deleted_at_idx` (`deleted_at_m`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `keys_permissions` (
  `temp_id` bigint NOT NULL AUTO_INCREMENT,
  `key_id` varchar(256) NOT NULL,
  `permission_id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`key_id`,`permission_id`,`workspace_id`),
  UNIQUE KEY `keys_permissions_temp_id_unique` (`temp_id`),
  UNIQUE KEY `key_id_permission_id_idx` (`key_id`,`permission_id`)
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `keys_roles` (
  `key_id` varchar(256) NOT NULL,
  `role_id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`role_id`,`key_id`,`workspace_id`),
  UNIQUE KEY `unique_key_id_role_id` (`key_id`,`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `permissions` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `name` varchar(512) NOT NULL,
  `description` varchar(512) DEFAULT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_name_per_workspace_idx` (`name`,`workspace_id`),
  KEY `workspace_id_idx` (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `quota` (
  `workspace_id` varchar(256) NOT NULL,
  `requests_per_month` bigint NOT NULL DEFAULT '0',
  `logs_retention_days` int NOT NULL DEFAULT '0',
  `audit_logs_retention_days` int NOT NULL DEFAULT '0',
  `team` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ratelimit_namespaces` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `name` varchar(512) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_name_per_workspace_idx` (`name`,`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ratelimit_overrides` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `namespace_id` varchar(256) NOT NULL,
  `identifier` varchar(512) NOT NULL,
  `limit` int NOT NULL,
  `duration` int NOT NULL,
  `async` tinyint(1) DEFAULT NULL,
  `sharding` enum('edge') DEFAULT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_identifier_per_namespace_idx` (`identifier`,`namespace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ratelimits` (
  `id` varchar(256) NOT NULL,
  `name` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint DEFAULT NULL,
  `key_id` varchar(256) DEFAULT NULL,
  `identity_id` varchar(256) DEFAULT NULL,
  `limit` int NOT NULL,
  `duration` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_name_idx` (`name`,`key_id`,`identity_id`),
  KEY `name_idx` (`name`),
  KEY `identity_id_idx` (`identity_id`),
  KEY `key_id_idx` (`key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `roles` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `name` varchar(512) NOT NULL,
  `description` varchar(512) DEFAULT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_name_per_workspace_idx` (`name`,`workspace_id`),
  KEY `workspace_id_idx` (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `roles_permissions` (
  `role_id` varchar(256) NOT NULL,
  `permission_id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`role_id`,`permission_id`,`workspace_id`),
  UNIQUE KEY `unique_tuple_permission_id_role_id` (`permission_id`,`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

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
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `project_environment_resource_type_idx` (`project_id`,`environment`,`resource_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `vercel_integrations` (
  `id` varchar(256) NOT NULL,
  `workspace_id` varchar(256) NOT NULL,
  `team_id` varchar(256) DEFAULT NULL,
  `access_token` varchar(256) NOT NULL,
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `workspaces` (
  `id` varchar(256) NOT NULL,
  `org_id` varchar(256) NOT NULL,
  `name` varchar(256) NOT NULL,
  `plan` enum('free','pro','enterprise') DEFAULT 'free',
  `tier` varchar(256) DEFAULT 'Free',
  `stripe_customer_id` varchar(256) DEFAULT NULL,
  `stripe_subscription_id` varchar(256) DEFAULT NULL,
  `beta_features` json NOT NULL,
  `features` json NOT NULL,
  `subscriptions` json DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `delete_protection` tinyint(1) DEFAULT '0',
  `created_at_m` bigint NOT NULL DEFAULT '0',
  `updated_at_m` bigint DEFAULT NULL,
  `deleted_at_m` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `workspaces_org_id_unique` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
