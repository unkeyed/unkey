-- runtime_logs_raw_v1: text-search index swap + ZSTD on LowCardinality columns.
--
-- 1. The old tokenbf_v1 indexes never matched the dashboard query's
--    positionCaseInsensitive(message, ...) substring predicate, so they were
--    dead weight. Replace with ngrambf_v1 over lower(...) and update the
--    dashboard query to call positionCaseInsensitive(lower(message), ...).
-- 2. Add CODEC(ZSTD(1)) to LowCardinality columns to compress the dictionary
--    pages (matches our String CODEC pattern).
--
-- After this migration is applied, run the following on prod to backfill the
-- new skip indexes against existing parts (otherwise they only cover new
-- inserts):
--
--   ALTER TABLE `default`.`runtime_logs_raw_v1` MATERIALIZE INDEX idx_message_text_search;
--   ALTER TABLE `default`.`runtime_logs_raw_v1` MATERIALIZE INDEX idx_attributes_text_search;

ALTER TABLE `default`.`runtime_logs_raw_v1`
    MODIFY COLUMN `severity` LowCardinality(String) CODEC(ZSTD(1));

ALTER TABLE `default`.`runtime_logs_raw_v1`
    MODIFY COLUMN `region` LowCardinality(String) CODEC(ZSTD(1));

ALTER TABLE `default`.`runtime_logs_raw_v1`
    MODIFY COLUMN `platform` LowCardinality(String) CODEC(ZSTD(1));

ALTER TABLE `default`.`runtime_logs_raw_v1` DROP INDEX IF EXISTS idx_message;
ALTER TABLE `default`.`runtime_logs_raw_v1` DROP INDEX IF EXISTS idx_attributes_text;

ALTER TABLE `default`.`runtime_logs_raw_v1`
    ADD INDEX IF NOT EXISTS idx_message_text_search lower(message)
    TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1;

ALTER TABLE `default`.`runtime_logs_raw_v1`
    ADD INDEX IF NOT EXISTS idx_attributes_text_search lower(attributes_text)
    TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1;
