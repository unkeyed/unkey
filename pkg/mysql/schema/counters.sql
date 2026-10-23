CREATE TABLE `counters` (
	`name` varchar(512) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`writer_id` varchar(48) COLLATE utf8mb4_0900_as_cs NOT NULL,
	`count` bigint NOT NULL DEFAULT 0,
	`revision` bigint unsigned NOT NULL,
	`delete_after` bigint,
	`compaction_after` bigint NOT NULL,
	`compaction_level` tinyint unsigned NOT NULL DEFAULT 0,
	CONSTRAINT `counters_name_writer_id_pk` PRIMARY KEY(`name`,`writer_id`)
);

CREATE INDEX `counter_delete_after` ON `counters` (`delete_after`);

CREATE INDEX `counter_compaction_after` ON `counters` (`compaction_after`);

