CREATE TABLE `app_source_oci` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`workspace_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`app_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`image_reference` varchar(512) NOT NULL,
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `app_source_oci_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `app_source_oci_app_id_idx` UNIQUE(`app_id`)
);
