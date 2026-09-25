CREATE TABLE `backoffice_audit_outbox` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`event_id` varchar(64) NOT NULL,
	`payload` json NOT NULL,
	`created_at` bigint NOT NULL,
	`drained_at` bigint unsigned,
	`attempts` int unsigned NOT NULL DEFAULT 0,
	`last_error` varchar(512),
	CONSTRAINT `backoffice_audit_outbox_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `backoffice_audit_outbox_event_id_unique` UNIQUE(`event_id`)
);

CREATE INDEX `drainer_pending_idx` ON `backoffice_audit_outbox` (`drained_at`,`pk`);

