-- key_verifications_raw_v2: switch columns from the LZ4 server default to
-- CODEC(ZSTD(1)).
--
-- These columns currently inherit the server default codec (LZ4). The table is
-- dominated by high-entropy ID columns (key_id, request_id), and ZSTD(1)
-- compresses them substantially better than LZ4. We use ZSTD(1) rather than
-- ZSTD(3): on this ID-heavy data higher levels give no benefit (measured
-- equal-to-worse) at higher insert-time CPU. Measured on 2M representative
-- rows: LZ4 -> ZSTD(1) is -46.4%.
--
-- MODIFY COLUMN only re-encodes parts written after this migration; existing
-- parts keep LZ4 until they are merged or dropped. On the S3-backed disk large
-- merges are disabled, so savings materialize as data rolls over the table's
-- 90-day TTL. Do NOT force it with OPTIMIZE ... FINAL: rewriting every part on
-- object storage is expensive and unnecessary here.

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `request_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `time` Int64 CODEC(Delta, ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `workspace_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `key_space_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `identity_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `external_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `key_id` String CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `region` LowCardinality(String) CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `source` LowCardinality(String) DEFAULT 'api' CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `outcome` LowCardinality(String) CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `tags` Array(String) DEFAULT [] CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `spent_credits` Int64 CODEC(ZSTD(1));

ALTER TABLE `default`.`key_verifications_raw_v2`
    MODIFY COLUMN `latency` Float64 CODEC(ZSTD(1));
