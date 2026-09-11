-- Runtime logs already have stable log IDs and server insertion timestamps.
-- Preserve timestamps, retention, and indexes when adding the drain cursor projection.
ALTER TABLE default.runtime_logs_raw_v1
    MODIFY SETTING allow_part_offset_column_in_projections = 1,
                   deduplicate_merge_projection_mode = 'rebuild';
ALTER TABLE default.runtime_logs_raw_v1
    ADD PROJECTION proj_logdrain
    (
        SELECT workspace_id, inserted_at, log_id, _part_offset
        ORDER BY workspace_id, inserted_at, log_id
    );
ALTER TABLE default.runtime_logs_raw_v1
    MATERIALIZE PROJECTION proj_logdrain SETTINGS mutations_sync = 1;
