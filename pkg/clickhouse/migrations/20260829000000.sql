-- Use the ClickHouse insertion time for audit log partitioning and retention.
-- Writers omit inserted_at so retries, delayed exports, and direct buffered
-- writes all receive the time when ClickHouse accepts the row.
ALTER TABLE `default`.`audit_logs_raw_v1`
    MODIFY COLUMN `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1));
