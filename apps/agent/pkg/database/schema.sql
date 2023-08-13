-- Create "apis" table
CREATE TABLE `apis` (
    `id` varchar(256) NOT NULL,
    `name` varchar(256) NOT NULL,
    `workspace_id` varchar(256) NOT NULL,
    `ip_whitelist` varchar(512) NULL,
    `auth_type` enum('key', 'jwt') NULL,
    `key_auth_id` varchar(256) NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `key_auth_id_idx` (`key_auth_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Create "key_auth" table
CREATE TABLE `key_auth` (
    `id` varchar(256) NOT NULL,
    `workspace_id` varchar(256) NOT NULL,
    PRIMARY KEY (`id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Create "keys" table
CREATE TABLE `keys` (
    `id` varchar(256) NOT NULL,
    `hash` varchar(256) NOT NULL,
    `start` varchar(256) NOT NULL,
    `owner_id` varchar(256) NULL,
    `meta` text NULL,
    `created_at` datetime(3) NOT NULL,
    `expires` datetime(3) NULL,
    `ratelimit_type` text NULL,
    `ratelimit_limit` int NULL,
    `ratelimit_refill_rate` int NULL,
    `ratelimit_refill_interval` int NULL,
    `workspace_id` varchar(256) NOT NULL,
    `for_workspace_id` varchar(256) NULL,
    `name` varchar(256) NULL,
    `remaining_requests` int NULL,
    `key_auth_id` varchar(256) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `hash_idx` (`hash`),
    INDEX `key_auth_id_idx` (`key_auth_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Create "workspaces" table
CREATE TABLE `workspaces` (
    `id` varchar(256) NOT NULL,
    `name` varchar(256) NOT NULL,
    `slug` varchar(256) NOT NULL,
    `tenant_id` varchar(256) NOT NULL,
    `internal` bool NOT NULL DEFAULT 0,
    `stripe_customer_id` varchar(256) NULL,
    `stripe_subscription_id` varchar(256) NULL,
    `plan` enum('free', 'pro', 'enterprise') NULL DEFAULT "free",
    PRIMARY KEY (`id`),
    UNIQUE INDEX `slug_idx` (`slug`),
    UNIQUE INDEX `tenant_id_idx` (`tenant_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;