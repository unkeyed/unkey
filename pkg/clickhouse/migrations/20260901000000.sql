-- Add an observation-only ClickHouse insertion timestamp to raw event tables
-- that can supply future log drains. Writers omit these columns, so the same
-- binaries are safe before and after this additive migration. Existing
-- partition and TTL expressions stay on event time. The timestamp is reliable
-- for rows inserted after this migration. We do not materialize old parts
-- because their original insertion time is unknown and rewriting the raw
-- tables would add avoidable load. ClickHouse can evaluate the default at read
-- or merge time for old parts, so those values do not record insertion time.

ALTER TABLE `default`.`key_verifications_raw_v2`
  ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1)) AFTER `time`;

ALTER TABLE `default`.`ratelimits_raw_v2`
  ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1)) AFTER `time`;

ALTER TABLE `default`.`api_requests_raw_v2`
  ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1)) AFTER `time`;

ALTER TABLE `default`.`frontline_requests_raw_v1`
  ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1)) AFTER `time`;
