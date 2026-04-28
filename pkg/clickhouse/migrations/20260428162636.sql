-- Add log_id to runtime_logs_raw_v1.
-- Minted by Vector at ingest time as "log_<16 hex chars>" (~64 bits entropy).
-- No backfill of pre-existing rows; they retain the default ''.
-- Not added to ORDER BY or any index — no query pattern uses it.
-- It exists as a stable join/key for the dashboard and downstream exporters.

ALTER TABLE `default`.`runtime_logs_raw_v1`
    ADD COLUMN IF NOT EXISTS `log_id` String DEFAULT '' CODEC(ZSTD(1)) AFTER `inserted_at`;
