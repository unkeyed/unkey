-- +goose up
CREATE TABLE ratelimits.ratelimits_last_used_v1
(
  time          Int64,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;



CREATE MATERIALIZED VIEW ratelimits.ratelimits_last_used_mv_v1
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


-- +goose down
DROP VIEW IF EXISTS ratelimits.ratelimits_last_used_mv_v1;
DROP TABLE IF EXISTS ratelimits.ratelimits_last_used_v1;
