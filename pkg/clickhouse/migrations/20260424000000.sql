-- Add per-row expires_at + dynamic TTL to the analytics raw tables.
--
-- Today these tables use a static `TTL ... INTERVAL N DELETE` which
-- gives every workspace the same retention regardless of plan. This
-- migration:
--   1. Adds an `expires_at Int64` column with a DEFAULT expression that
--      reproduces the table's existing static window (per-table — they
--      differ: verifications 90d, ratelimits/api_requests 30d, sentinel
--      30d).
--   2. Replaces the static TTL with one that reads `expires_at`.
--
-- Behavior is identical for any row inserted today: the writer doesn't
-- send `expires_at`, CH applies DEFAULT, the row expires on the same
-- schedule as before. Once writers are updated to stamp `expires_at`
-- from the workspace's `logs_retention_days` quota, per-workspace
-- retention takes over without further schema changes.
--
-- Existing parts keep their original TTL metadata until they're merged
-- (CH reapplies TTL on merge), so retention behavior on already-stored
-- data shifts gradually as merges happen — acceptable since the new
-- DEFAULT matches the old static value.
--
-- runtime_logs_raw_v1 already has its own `expires_at DateTime64(3)` +
-- TTL clause; not touched here.

-- key_verifications_raw_v2: was `time + INTERVAL 90 DAY` (7776000000 ms)
ALTER TABLE default.key_verifications_raw_v2
    ADD COLUMN IF NOT EXISTS `expires_at` Int64 DEFAULT time + 7776000000 CODEC(Delta, LZ4);

ALTER TABLE default.key_verifications_raw_v2
    MODIFY TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE;

-- ratelimits_raw_v2: was `time + INTERVAL 1 MONTH` (~30d, 2592000000 ms)
ALTER TABLE default.ratelimits_raw_v2
    ADD COLUMN IF NOT EXISTS `expires_at` Int64 DEFAULT time + 2592000000 CODEC(Delta, LZ4);

ALTER TABLE default.ratelimits_raw_v2
    MODIFY TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE;

-- api_requests_raw_v2: was `time + INTERVAL 1 MONTH` (~30d)
ALTER TABLE default.api_requests_raw_v2
    ADD COLUMN IF NOT EXISTS `expires_at` Int64 DEFAULT time + 2592000000 CODEC(Delta, LZ4);

ALTER TABLE default.api_requests_raw_v2
    MODIFY TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE;

-- sentinel_requests_raw_v1: was `time + toIntervalDay(30)` (30d)
ALTER TABLE default.sentinel_requests_raw_v1
    ADD COLUMN IF NOT EXISTS `expires_at` Int64 DEFAULT time + 2592000000 CODEC(Delta, LZ4);

ALTER TABLE default.sentinel_requests_raw_v1
    MODIFY TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE;
