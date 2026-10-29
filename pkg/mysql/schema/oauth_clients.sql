CREATE TABLE `oauth_clients` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`name` varchar(256) NOT NULL,
	`redirect_uris` json NOT NULL,
	`metadata_hash` varchar(64) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`registrar_ip` varchar(64),
	`registrar_user_agent` varchar(512),
	`created_at` bigint NOT NULL,
	CONSTRAINT `oauth_clients_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `oauth_clients_id_unique` UNIQUE(`id`),
	CONSTRAINT `oauth_clients_metadata_hash_unique` UNIQUE(`metadata_hash`)
);

