ALTER TABLE `default`.`key_verifications_raw_v2` ADD COLUMN `external_id` String;
ALTER TABLE `default`.`key_verifications_raw_v2` ADD INDEX `idx_external_id` ((external_id)) TYPE bloom_filter GRANULARITY 1;
