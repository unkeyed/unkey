-- Temporary materialized view to sync new writes from v1 to v2 during migration
-- This ensures zero-downtime migration by duplicating all new inserts
-- DROP this view after migration is complete and application switches to v2

CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.temp_sync_v1_to_v2 
TO verifications.raw_key_verifications_v2 
AS
SELECT 
    request_id,
    time,
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    region,
    outcome,
    tags,
    0 as spent_credits,    -- v1 doesn't have this column, default to 0
    0.0 as latency         -- v1 doesn't have this column, default to 0.0
FROM verifications.raw_key_verifications_v1;