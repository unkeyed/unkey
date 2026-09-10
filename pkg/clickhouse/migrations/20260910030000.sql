-- Enable drains only after every API writer supplies check_index and this migration completes.
-- Old multi-limit rows have no check index and are excluded by their frozen insertion time.
ALTER TABLE default.ratelimits_raw_v2
    ADD COLUMN check_index UInt32 DEFAULT 0 AFTER request_id,
    ADD COLUMN event_id String MATERIALIZED concat(request_id, ':', toString(check_index)) AFTER check_index,
    ADD COLUMN inserted_at Int64 DEFAULT 0 CODEC(Delta, ZSTD(1)) AFTER time;
ALTER TABLE default.ratelimits_raw_v2
    MATERIALIZE COLUMN inserted_at SETTINGS mutations_sync = 1;
ALTER TABLE default.ratelimits_raw_v2
    MODIFY COLUMN inserted_at Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1));

ALTER TABLE default.ratelimits_raw_v2
    MODIFY SETTING allow_part_offset_column_in_projections = 1,
                   deduplicate_merge_projection_mode = 'rebuild';
ALTER TABLE default.ratelimits_raw_v2
    ADD PROJECTION proj_logdrain
    (
        SELECT workspace_id, inserted_at, event_id, _part_offset
        ORDER BY workspace_id, inserted_at, event_id
    );
ALTER TABLE default.ratelimits_raw_v2
    MATERIALIZE PROJECTION proj_logdrain SETTINGS mutations_sync = 1;
