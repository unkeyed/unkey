-- +goose up
CREATE TABLE telemetry.raw_sdks_v1(
  -- the api request id, so we can correlate the telemetry with traces and logs
  request_id String,

  -- unix milli
  time Int64,

  -- ie: node@20
  runtime String,
  -- ie: vercel
  platform String,

  -- ie: [ "@unkey/api@1.2.3", "@unkey/ratelimit@4.5.6" ]
  versions Array(String)
)
ENGINE = MergeTree()
ORDER BY (request_id, time)
;



-- +goose down
DROP TABLE telemetry.raw_sdks_v1;
