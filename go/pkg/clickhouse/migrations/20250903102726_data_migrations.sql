-- Create "temp_sync_key_verifications_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_key_verifications_v1_to_v2` TO `default`.`key_verifications_raw_v2` AS SELECT request_id, time, workspace_id, key_space_id, identity_id, key_id, region, outcome, tags, 0 AS spent_credits, 0. AS latency FROM verifications.raw_key_verifications_v1;
-- Create "temp_sync_ratelimits_raw_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2` TO `default`.`ratelimits_raw_v2` AS SELECT request_id, time, workspace_id, namespace_id, identifier, passed, 0. AS latency FROM ratelimits.raw_ratelimits_v1;
