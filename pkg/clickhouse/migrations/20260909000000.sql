ALTER TABLE `default`.`audit_logs_raw_v1`
    MODIFY SETTING allow_part_offset_column_in_projections = 1,
                   deduplicate_merge_projection_mode = 'rebuild';

ALTER TABLE `default`.`audit_logs_raw_v1`
    ADD PROJECTION IF NOT EXISTS proj_logdrain
    (
        SELECT workspace_id, inserted_at, event_id, _part_offset
        ORDER BY workspace_id, inserted_at, event_id
    );

ALTER TABLE `default`.`audit_logs_raw_v1`
    MATERIALIZE PROJECTION proj_logdrain SETTINGS mutations_sync = 1;

ALTER TABLE `default`.`audit_logs_raw_v1`
    DROP INDEX IF EXISTS idx_inserted_at;
