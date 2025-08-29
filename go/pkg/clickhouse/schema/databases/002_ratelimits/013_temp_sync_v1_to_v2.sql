-- Temporary materialized view to sync new writes from v1 to v2 during migration
-- This ensures zero-downtime migration by duplicating all new inserts
-- DROP this view after migration is complete and application switches to v2

CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.temp_sync_v1_to_v2 
TO ratelimits.raw_ratelimits_v2 
AS
SELECT 
    request_id,
    time,
    workspace_id,
    namespace_id,
    identifier,
    passed,
    0.0 as latency         -- v1 doesn't have this column, default to 0.0
FROM ratelimits.raw_ratelimits_v1;