-- +goose up
ALTER TABLE metrics.raw_api_requests_v1
    ADD INDEX idx_workspace_time (workspace_id, time) TYPE minmax GRANULARITY 1;

ALTER TABLE metrics.raw_api_requests_v1
    ADD INDEX idx_request_id (request_id) TYPE minmax GRANULARITY 1;

-- +goose down
ALTER TABLE metrics.raw_api_requests_v1
    DROP INDEX idx_workspace_time;

ALTER TABLE metrics.raw_api_requests_v1
    DROP INDEX idx_request_id;
