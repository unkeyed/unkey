-- Billable passed ratelimit checks per end-user identity per month, KEYED BY
-- NAMESPACE. Supersedes billable_ratelimits_per_identity_per_month_v1 by
-- retaining namespace_id in the grain so billing can be scoped to the ratelimit
-- namespaces a customer has enabled (billing_billable_resources).
-- Reads the attributed raw v3 stream: each raw insert contributes exactly once,
-- and the SummingMergeTree collapses to one quantity per key. The raw table
-- TTLs at 1 month, but MV rows written here persist beyond it — so monthly
-- billable quantities survive; raw-level reconstruction does not
-- (final-at-period-close, per the plan's retention caveat).
-- Unattributed rows (empty external_id) are excluded.
CREATE TABLE billable_ratelimits_per_identity_per_month_v2 (
  year Int16,
  month Int8,
  workspace_id String,
  namespace_id String,
  identity_id String,
  external_id String,
  count Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (workspace_id, namespace_id, external_id, identity_id, year, month);

CREATE MATERIALIZED VIEW billable_ratelimits_per_identity_per_month_mv_v2 TO billable_ratelimits_per_identity_per_month_v2 AS
SELECT
  workspace_id,
  namespace_id,
  identity_id,
  external_id,
  toInt64 (countIf (passed)) AS count,
  toYear (fromUnixTimestamp64Milli (time)) AS year,
  toMonth (fromUnixTimestamp64Milli (time)) AS month
FROM
  ratelimits_raw_v3
WHERE
  external_id != ''
GROUP BY
  workspace_id,
  namespace_id,
  identity_id,
  external_id,
  year,
  month;
