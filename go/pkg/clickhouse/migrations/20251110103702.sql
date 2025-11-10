-- Fix materialized view temp_sync_key_verifications_v1_to_v2 to include external_id column
-- The original migration was missing the external_id column and had wrong column order

-- Drop the incorrect materialized view
DROP TABLE IF EXISTS `default`.`temp_sync_key_verifications_v1_to_v2`;

-- Recreate with correct column order including external_id
CREATE MATERIALIZED VIEW `default`.`temp_sync_key_verifications_v1_to_v2` TO `default`.`key_verifications_raw_v2` AS 
SELECT 
    request_id, 
    time, 
    workspace_id, 
    key_space_id, 
    identity_id, 
    '' as external_id, 
    key_id, 
    region, 
    outcome, 
    tags, 
    0 as spent_credits, 
    0.0 as latency 
FROM verifications.raw_key_verifications_v1;