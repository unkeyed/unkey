;
CREATE TABLE IF NOT EXISTS ratelimits_last_used_v2
(
  time          Int64,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, identifier)
;



CREATE MATERIALIZED VIEW IF NOT EXISTS ratelimits_last_used_mv_v2
TO ratelimits.ratelimits_last_used_v2
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
