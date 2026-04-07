CREATE TABLE `acme_challenges` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`domain_id` varchar(255) NOT NULL,
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

CREATE INDEX `workspace_idx` ON `acme_challenges` (`workspace_id`);

CREATE INDEX `status_idx` ON `acme_challenges` (`status`);

