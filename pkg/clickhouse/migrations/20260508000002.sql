ALTER TABLE `default`.`audit_logs_raw_v1` MODIFY TTL toDateTime(fromUnixTimestamp64Milli(inserted_at)) + toIntervalDay(90);
