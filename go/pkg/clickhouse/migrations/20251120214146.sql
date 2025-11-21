ALTER TABLE `default`.`key_verifications_per_day_v3` MODIFY TTL time + toIntervalDay(356);
ALTER TABLE `default`.`key_verifications_per_hour_v3` MODIFY TTL time + toIntervalDay(90);
ALTER TABLE `default`.`key_verifications_per_minute_v3` MODIFY TTL time + toIntervalDay(90);
ALTER TABLE `default`.`key_verifications_raw_v2` MODIFY TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90);
