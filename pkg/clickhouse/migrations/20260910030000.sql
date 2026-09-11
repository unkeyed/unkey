ALTER TABLE default.ratelimits_raw_v2
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
        SELECT workspace_id, inserted_at, request_id, _part_offset
        ORDER BY workspace_id, inserted_at, request_id
    );
ALTER TABLE default.ratelimits_raw_v2
    MATERIALIZE PROJECTION proj_logdrain SETTINGS mutations_sync = 1;
