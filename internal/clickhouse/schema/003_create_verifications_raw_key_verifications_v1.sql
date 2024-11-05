-- +goose up
CREATE TABLE verifications.raw_key_verifications_v1(
    -- the api request id, so we can correlate the verification with traces and logs
    request_id String,

    -- unix milli
    time Int64,

    workspace_id String,
    key_space_id String,
    key_id String,

    -- Right now this is a 3 character airport code, but when we move to aws,
    -- this will be the region code such as `us-east-1`
    region LowCardinality(String),

    -- Examples:
    -- - "VALID"
    -- - "RATE_LIMITED"
    -- - "EXPIRED"
    -- - "DISABLED
    outcome LowCardinality(String),

    -- Empty string if the key has no identity
    identity_id String,



)
ENGINE = MergeTree()
ORDER BY (workspace_id, key_space_id, key_id, time)
;

-- +goose down
DROP TABLE verifications.raw_key_verifications_v1;
