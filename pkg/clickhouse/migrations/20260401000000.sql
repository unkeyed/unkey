ALTER TABLE `default`.`sentinel_requests_raw_v1` ADD COLUMN `platform` LowCardinality(String) AFTER `region`;
ALTER TABLE `default`.`runtime_logs_raw_v1` ADD COLUMN `platform` LowCardinality(String) AFTER `region`;
