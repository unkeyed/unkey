CREATE TABLE `portal_branding` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`portal_config_id` varchar(64) NOT NULL,
	`logo_url` varchar(500),
	`primary_color` varchar(7),
	`secondary_color` varchar(7),
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `portal_branding_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `portal_branding_portal_config_id_unique` UNIQUE(`portal_config_id`)
);

