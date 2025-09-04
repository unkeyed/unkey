CREATE DATABASE IF NOT EXISTS verifications;

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

    tags Array(String) DEFAULT []



)
ENGINE = MergeTree()
ORDER BY (workspace_id, key_space_id, key_id, time)
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_hour_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, time, identity_id, key_id)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_hour_mv_v1
TO verifications.key_verifications_per_hour_v1
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfHour(fromUnixTimestamp64Milli(time)) AS time
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_day_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, time, identity_id, key_id)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_day_mv_v1
TO verifications.key_verifications_per_day_v1
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfDay(fromUnixTimestamp64Milli(time)) AS time
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_month_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, time, identity_id, key_id)
;

CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_month_mv_v1
TO verifications.key_verifications_per_month_v1
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_hour_v2
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_hour_mv_v2
TO verifications.key_verifications_per_hour_v2
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfHour(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_day_v2
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_day_mv_v2
TO verifications.key_verifications_per_day_v2
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfDay(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_month_v2
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_month_mv_v2
TO verifications.key_verifications_per_month_v2
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_hour_v3
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags, outcome)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_hour_mv_v3
TO verifications.key_verifications_per_hour_v3
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfHour(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_day_v3
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags, outcome)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_day_mv_v3
TO verifications.key_verifications_per_day_v3
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfDay(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_month_v3
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags, outcome)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_month_mv_v3
TO verifications.key_verifications_per_month_v3
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE TABLE verifications.key_verifications_per_minute_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, identity_id, key_id, time, tags, outcome)
;


CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_minute_mv_v1
TO verifications.key_verifications_per_minute_v1
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;
CREATE DATABASE IF NOT EXISTS ratelimits;
CREATE TABLE IF NOT EXISTS ratelimits.raw_ratelimits_v1(
    request_id    String,
    -- unix milli
    time          Int64,
    workspace_id  String,
    namespace_id  String,
    identifier    String,
    passed        Bool

)
ENGINE = MergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)

;

ALTER TABLE ratelimits.raw_ratelimits_v1 ADD INDEX IF NOT EXISTS idx_workspace_time (workspace_id, time) TYPE minmax GRANULARITY 1;

ALTER TABLE ratelimits.raw_ratelimits_v1
ADD INDEX IF NOT EXISTS idx_request_id (request_id) TYPE minmax GRANULARITY 1;
CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_per_minute_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  passed        Int64,
  total         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_per_minute_mv_v1
TO ratelimits.ratelimits_per_minute_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  countIf(passed > 0) as passed,
  count(*) as total,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;
CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_per_hour_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  passed        Int64,
  total         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_per_hour_mv_v1
TO ratelimits.ratelimits_per_hour_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  countIf(passed > 0) as passed,
  count(*) as total,
  toStartOfHour(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;
CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_per_day_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  passed        Int64,
  total         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_per_day_mv_v1
TO ratelimits.ratelimits_per_day_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  count(*) as total,
  countIf(passed > 0) as passed,
  toStartOfDay(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;
CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_per_month_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  passed        Int64,
  total         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_per_month_mv_v1
TO ratelimits.ratelimits_per_month_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  countIf(passed > 0) as passed,
  count(*) as total,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier,
  time
;
CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_last_used_v1
(
  time          Int64,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;



CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits.ratelimits_last_used_mv_v1
TO ratelimits.ratelimits_last_used_v1
AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  maxSimpleState(time) as time
FROM ratelimits.raw_ratelimits_v1
GROUP BY
  workspace_id,
  namespace_id,
  identifier
;
CREATE DATABASE IF NOT EXISTS metrics;
CREATE TABLE IF NOT EXISTS metrics.raw_api_requests_v1(
    request_id String,
    -- unix milli
    time Int64,

    workspace_id String,

    host String,

    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    path String,
    -- "Key: Value" pairs
    request_headers Array(String),
    request_body String,

    response_status Int,
    -- "Key: Value" pairs
    response_headers Array(String),
    response_body String,
    -- internal err.Error() string, empty if no error
    error String,

    -- milliseconds
    service_latency Int64,

    user_agent String,
    ip_address String,
    country String,
    city String,
    colo String,
    continent String,


)
ENGINE = MergeTree()
ORDER BY (workspace_id, time, request_id)
;

ALTER TABLE metrics.raw_api_requests_v1
    ADD INDEX IF NOT EXISTS idx_workspace_time (workspace_id, time) TYPE minmax GRANULARITY 1;

ALTER TABLE metrics.raw_api_requests_v1
    ADD INDEX IF NOT EXISTS idx_request_id (request_id) TYPE minmax GRANULARITY 1;
CREATE TABLE IF NOT EXISTS metrics.api_requests_per_hour_v1 (
    time DateTime,
    workspace_id String,
    path String,
    response_status Int,
    host String,
    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY
    (
        workspace_id,
        time,
        host,
        path,
        response_status,
        method
    );
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics.api_requests_per_hour_mv_v1 TO metrics.api_requests_per_hour_v1 AS
SELECT
    workspace_id,
    path,
    response_status,
    host,
    method,
    count(*) as count,
    toStartOfHour(fromUnixTimestamp64Milli(time)) AS time
FROM
    metrics.raw_api_requests_v1
GROUP BY
    workspace_id,
    path,
    response_status,
    host,
    method,
    time;
CREATE TABLE IF NOT EXISTS metrics.api_requests_per_minute_v1 (
    time DateTime,
    workspace_id String,
    path String,
    response_status Int,
    host String,
    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY
    (
        workspace_id,
        time,
        host,
        path,
        response_status,
        method
    );
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics.api_requests_per_minute_mv_v1 TO metrics.api_requests_per_minute_v1 AS
SELECT
    workspace_id,
    path,
    response_status,
    host,
    method,
    count(*) as count,
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM
    metrics.raw_api_requests_v1
GROUP BY
    workspace_id,
    path,
    response_status,
    host,
    method,
    time;
CREATE TABLE IF NOT EXISTS metrics.api_requests_per_day_v1 (
    time DateTime,
    workspace_id String,
    path String,
    response_status Int,
    host String,
    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY
    (
        workspace_id,
        time,
        host,
        path,
        response_status,
        method
    );


CREATE MATERIALIZED VIEW IF NOT EXISTS metrics.api_requests_per_day_mv_v1 TO metrics.api_requests_per_day_v1 AS
SELECT
    workspace_id,
    path,
    response_status,
    host,
    method,
    count(*) as count,
    toStartOfDay(fromUnixTimestamp64Milli(time)) AS time
FROM
    metrics.raw_api_requests_v1
GROUP BY
    workspace_id,
    path,
    response_status,
    host,
    method,
    time;
CREATE DATABASE IF NOT EXISTS billing;
CREATE DATABASE IF NOT EXISTS verifications;
CREATE DATABASE IF NOT EXISTS ratelimits;
CREATE TABLE IF NOT EXISTS billing.billable_verifications_per_month_v1
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS billing.billable_verifications_per_month_mv_v1
TO billing.billable_verifications_per_month_v1
AS
SELECT
  workspace_id,
  count(*) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM verifications.key_verifications_per_month_v2
WHERE outcome = 'VALID'
GROUP BY
  workspace_id,
  year,
  month
;
CREATE TABLE IF NOT EXISTS billing.billable_verifications_per_month_v2
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;

CREATE MATERIALIZED VIEW IF NOT EXISTS billing.billable_verifications_per_month_mv_v2
TO billing.billable_verifications_per_month_v2
AS SELECT
    workspace_id,
    sum(count) AS count,
    toYear(time) AS year,
    toMonth(time) AS month
FROM verifications.key_verifications_per_month_v1
WHERE outcome = 'VALID'
GROUP BY
    workspace_id,
    year,
    month
;
CREATE TABLE IF NOT EXISTS billing.billable_ratelimits_per_month_v1
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;

CREATE MATERIALIZED VIEW IF NOT EXISTS billing.billable_ratelimits_per_month_mv_v1
TO billing.billable_ratelimits_per_month_v1
AS SELECT
    workspace_id,
    sum(passed) AS count,
    toYear(time) AS year,
    toMonth(time) AS month
FROM ratelimits.ratelimits_per_month_v1
WHERE passed > 0
GROUP BY
    workspace_id,
    year,
    month
;
CREATE DATABASE IF NOT EXISTS business;
CREATE DATABASE IF NOT EXISTS billing;
CREATE TABLE IF NOT EXISTS business.active_workspaces_per_month_v1
(
  time          Date,
  workspace_id  String
)
ENGINE = MergeTree()
ORDER BY (time)
;
CREATE MATERIALIZED VIEW IF NOT EXISTS business.active_workspaces_keys_per_month_mv_v1
TO business.active_workspaces_per_month_v1
AS
SELECT
  workspace_id, toDate(time) as time
FROM verifications.key_verifications_per_month_v2
;

CREATE MATERIALIZED VIEW IF NOT EXISTS business.active_workspaces_ratelimits_per_month_mv_v1
TO business.active_workspaces_per_month_v1
AS
SELECT
  workspace_id, toDate(time) as time
FROM ratelimits.ratelimits_per_month_v1
;
