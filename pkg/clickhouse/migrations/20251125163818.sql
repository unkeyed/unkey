ALTER TABLE `default`.`key_verifications_per_day_v3` MODIFY TTL time + toIntervalDay(365);
