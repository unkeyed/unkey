-- api_requests_raw_v2: switch text/blob columns from the LZ4 server default to
-- CODEC(ZSTD(3)).
--
-- The table stores full request/response bodies and both header sets verbatim
-- (they are shown in the dashboard, so we cannot truncate them), which is the
-- bulk of its on-disk size. These columns currently inherit the server default
-- codec (LZ4). ZSTD is lossless, so full bodies are preserved while compressing
-- far better than LZ4 on repetitive JSON.
--
-- We use ZSTD(3) rather than our usual ZSTD(1) String default: this table is
-- dominated by large bodies that live on S3 and are read infrequently, so the
-- extra compression is worth the higher insert-time CPU. Measured on 500k
-- representative rows: LZ4 -> ZSTD(1) is -41.5%, ZSTD(3) is -43.8%.
--
-- MODIFY COLUMN only re-encodes parts written after this migration; existing
-- parts keep LZ4 until they are merged or dropped. On the S3-backed disk large
-- merges are disabled, so savings materialize as data rolls over the table's
-- 1-month TTL. Do NOT force it with OPTIMIZE ... FINAL: rewriting every part on
-- object storage is expensive and unnecessary here.

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `request_id` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `time` Int64 CODEC(Delta, ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `workspace_id` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `host` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `method` LowCardinality(String) CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `path` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `query_string` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `query_params` Map(String, Array(String)) CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `request_headers` Array(String) CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `request_body` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `response_headers` Array(String) CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `response_body` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `error` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `user_agent` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `ip_address` String CODEC(ZSTD(3));

ALTER TABLE `default`.`api_requests_raw_v2`
    MODIFY COLUMN `region` LowCardinality(String) CODEC(ZSTD(3));
