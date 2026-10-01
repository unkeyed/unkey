ALTER TABLE default.build_step_logs_v1
    ADD COLUMN seq UInt64 DEFAULT toUInt64(time) * 1000 CODEC(Delta, ZSTD),
    ADD COLUMN error Bool DEFAULT false,
    ADD INDEX idx_seq seq TYPE minmax GRANULARITY 1;
