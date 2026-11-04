-- Add the workspace-scoped key fingerprint to key verification failures.
--
-- key_fingerprint is the first 8 bytes of
-- HMAC-SHA256(secret, workspace_id || 0x00 || key hash). One key has a
-- different fingerprint in every workspace. It is 0 when svc/api has no
-- fingerprint secret.
--
-- The rollup gains key_fingerprints, the distinct submitted keys per minute.
-- Rows written before this migration have no fingerprint, so those minutes
-- count no distinct keys.
--
-- DEPLOYMENT ORDER: apply this migration before deploying writers that include
-- key_fingerprint in their insert column list. Old writers remain compatible
-- because the raw column has a server-side default.

ALTER TABLE `default`.`key_verification_failures_raw_v1`
  ADD COLUMN `key_fingerprint` UInt64 DEFAULT 0 AFTER `key_workspace_id`,
  ADD INDEX `idx_key_fingerprint` key_fingerprint TYPE bloom_filter GRANULARITY 4;

ALTER TABLE `default`.`key_verification_failures_per_minute_v1`
  ADD COLUMN `key_fingerprints` AggregateFunction(uniqCombined64, UInt64) AFTER `cache_misses`;

ALTER TABLE `default`.`key_verification_failures_per_minute_mv_v1` MODIFY QUERY
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id,
    root_key_id,
    outcome,
    reason,
    region,
    source,
    toUInt64(sum(sample_weight)) AS count,
    toUInt64(sumIf(sample_weight, cache_result = 'miss')) AS cache_misses,
    uniqCombined64State(key_fingerprint) AS key_fingerprints,
    uniqCombined64State(ip_address) AS ip_addresses,
    topKState(5)(key_length) AS key_lengths,
    topKState(5)(toString(key_prefix)) AS key_prefixes
FROM `default`.`key_verification_failures_raw_v1`
GROUP BY
    time,
    workspace_id,
    root_key_id,
    outcome,
    reason,
    region,
    source;
