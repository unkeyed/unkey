CREATE TABLE `log_drain_credentials` (
	`drain_id` varchar(64) NOT NULL,
	`source` enum('paste','oauth') NOT NULL,
	`encrypted_credentials` varchar(1024),
	`encryption_key_id` varchar(256),
	`oauth_grant_id` varchar(64),
	`updated_at` bigint NOT NULL,
	CONSTRAINT `log_drain_credentials_drain_id` PRIMARY KEY(`drain_id`)
);

