-- Add correlation_id to audit_logs_raw_v1.
--
-- Groups events emitted by one logical user action so the dashboard can
-- show "everything that happened when this key was created" without
-- timestamp-window guessing. Auto-minted by the audit log Insert service
-- when a caller batches >1 events; opt-in via
-- auditlog.WithCorrelation(ctx, ...) for flows that fan out across
-- multiple Insert calls (v2/keys.createKey -> withPermissions ->
-- withRoles). Empty on single-event flows that don't need grouping.
--
-- Defaults to "" (CH String). All existing call sites continue to work
-- unchanged.

ALTER TABLE default.audit_logs_raw_v1
    ADD COLUMN IF NOT EXISTS `correlation_id` String CODEC(ZSTD(1));

-- Bloom filter sized for ~1% FP rate so "find all events in this user
-- action" stays a single-granule lookup. Sparse on empty correlation_id
-- (no false positives because empty hashes deterministically).
ALTER TABLE default.audit_logs_raw_v1
    ADD INDEX IF NOT EXISTS idx_correlation_id correlation_id TYPE bloom_filter(0.01) GRANULARITY 4;
