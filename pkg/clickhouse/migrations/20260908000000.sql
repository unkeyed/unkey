-- Record the ClickHouse insertion time on the request tables so log drains
-- can page by a clock that never lags behind buffered or retried inserts.
-- Writers omit inserted_at and let the server default stamp the row.
-- Existing rows receive 0, not a backfilled clock, because MATERIALIZE
-- COLUMN would stamp them with the migration time instead of their real
-- insertion time.
ALTER TABLE `default`.`api_requests_raw_v2`
    ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(3)) AFTER `time`;
ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, LZ4) AFTER `time`;
