-- +goose up
ALTER TABLE ratelimits.raw_ratelimits_v1
    ADD INDEX idx_workspace_time (workspace_id, time) TYPE minmax GRANULARITY 1;

ALTER TABLE ratelimits.raw_ratelimits_v1
    ADD INDEX idx_request_id (request_id) TYPE minmax GRANULARITY 1;

-- +goose down
ALTER TABLE ratelimits.raw_ratelimits_v1
    DROP INDEX idx_workspace_time;

ALTER TABLE ratelimits.raw_ratelimits_v1
    DROP INDEX idx_request_id;
