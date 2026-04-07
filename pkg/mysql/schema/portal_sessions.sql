CREATE TABLE `portal_sessions` (
        `pk` bigint unsigned AUTO_INCREMENT NOT NULL,
        `id` varchar(64) NOT NULL,
        `workspace_id` varchar(256) NOT NULL,
        `portal_config_id` varchar(64) NOT NULL,
        `external_id` varchar(256) NOT NULL,
        `metadata` json,
        `permissions` json NOT NULL,
        `preview` boolean NOT NULL DEFAULT false,
        `expires_at` bigint NOT NULL,
        `created_at` bigint NOT NULL,
        CONSTRAINT `portal_sessions_pk` PRIMARY KEY(`pk`),
        CONSTRAINT `portal_sessions_id_unique` UNIQUE(`id`)
);
