-- Requires the insertion timestamp migration 20260908000000.sql.
-- Materialize before enabling gateway request drains. Existing timestamps,
-- the base sorting key, indexes, and event-time retention remain unchanged.
ALTER TABLE default.frontline_requests_raw_v1
    MODIFY SETTING allow_part_offset_column_in_projections = 1,
                   deduplicate_merge_projection_mode = 'rebuild';
ALTER TABLE default.frontline_requests_raw_v1
    ADD PROJECTION proj_logdrain
    (
        SELECT workspace_id, inserted_at, request_id, _part_offset
        ORDER BY workspace_id, inserted_at, request_id
    );
ALTER TABLE default.frontline_requests_raw_v1
    MATERIALIZE PROJECTION proj_logdrain SETTINGS mutations_sync = 1;
