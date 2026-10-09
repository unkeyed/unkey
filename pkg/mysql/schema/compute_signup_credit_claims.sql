CREATE TABLE `compute_signup_credit_claims` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`card_fingerprint` varchar(128) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`stripe_customer_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`stripe_balance_transaction_id` varchar(256) COLLATE utf8mb4_0900_as_cs,
	`workos_user_id` varchar(256) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`attempt_id` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`created_at` bigint NOT NULL,
	CONSTRAINT `compute_signup_credit_claims_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `compute_signup_credit_claims_fingerprint_uq` UNIQUE(`card_fingerprint`),
	CONSTRAINT `compute_signup_credit_claims_workspace_uq` UNIQUE(`workspace_id`),
	CONSTRAINT `compute_signup_credit_claims_user_uq` UNIQUE(`workos_user_id`)
);

