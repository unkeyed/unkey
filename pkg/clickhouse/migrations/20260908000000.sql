-- Record the ClickHouse insertion time on the request tables so log drains
-- can page by a clock that never lags behind buffered or retried inserts.
-- Writers omit inserted_at and let the server default stamp the row.
-- Existing rows receive 0, not a backfilled clock, because MATERIALIZE
-- COLUMN would stamp them with the migration time instead of their real
-- insertion time.
ALTER TABLE `default`.`api_requests_raw_v2`
    ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(3)) AFTER `time`;
ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(3)) AFTER `time`;

-- inserted_at is not in either sorting key. A minmax skip index prunes
-- granules outside a drain's insertion time window because rows in a granule
-- were inserted within seconds of each other. MATERIALIZE INDEX builds it for
-- existing parts, where every row reads inserted_at as 0, so those parts are
-- skipped by any window that starts after this migration.
ALTER TABLE `default`.`api_requests_raw_v2`
    ADD INDEX `idx_inserted_at` inserted_at TYPE minmax GRANULARITY 1;
ALTER TABLE `default`.`api_requests_raw_v2`
    MATERIALIZE INDEX `idx_inserted_at`;
ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX `idx_inserted_at` inserted_at TYPE minmax GRANULARITY 1;
ALTER TABLE `default`.`frontline_requests_raw_v1`
    MATERIALIZE INDEX `idx_inserted_at`;

-- The audit and runtime tables already page by inserted_at but only prune
-- by partition (month and day). The same minmax index prunes granules
-- inside a partition. Existing parts hold real insertion times, so
-- materializing gives them the full benefit.
ALTER TABLE `default`.`runtime_logs_raw_v1`
    ADD INDEX `idx_inserted_at` inserted_at TYPE minmax GRANULARITY 1;
ALTER TABLE `default`.`runtime_logs_raw_v1`
    MATERIALIZE INDEX `idx_inserted_at`;
ALTER TABLE `default`.`audit_logs_raw_v1`
    ADD INDEX `idx_inserted_at` inserted_at TYPE minmax GRANULARITY 1;
ALTER TABLE `default`.`audit_logs_raw_v1`
    MATERIALIZE INDEX `idx_inserted_at`;
