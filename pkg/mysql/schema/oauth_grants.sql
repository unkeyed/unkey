CREATE TABLE `oauth_grants` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(64) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`provider` varchar(64) NOT NULL,
	`account_label` varchar(256) NOT NULL,
	`region` varchar(32),
	`scopes` json NOT NULL DEFAULT ('[]'),
	`encrypted_credentials` varchar(1024) NOT NULL,
	`encryption_key_id` varchar(256) NOT NULL,
	`expires_at` bigint,
	`revoked_at` bigint,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `oauth_grants_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `oauth_grants_id_unique` UNIQUE(`id`)
);

CREATE INDEX `oauth_grants_workspace_idx` ON `oauth_grants` (`workspace_id`,`provider`,`revoked_at`);

