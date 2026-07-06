-- Billable verifications per end-user identity per month, KEYED BY KEYSPACE.
-- Supersedes billable_verifications_per_identity_per_month_v1 by retaining
-- key_space_id in the grain, so end-user billing can be scoped to only the
-- keyspaces a customer has enabled (billing_billable_resources). Rows with an
-- empty external_id (no identity attached) are excluded rather than attributed
-- to a blank subject; workspace-grained billing remains the authority for
-- totals.
CREATE TABLE billable_verifications_per_identity_per_month_v2 (
  year Int16,
  month Int8,
  workspace_id String,
  key_space_id String,
  identity_id String,
  external_id String,
  count Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (workspace_id, key_space_id, external_id, identity_id, year, month);

CREATE MATERIALIZED VIEW billable_verifications_per_identity_per_month_mv_v2 TO billable_verifications_per_identity_per_month_v2 AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  sum(count) AS count,
  toYear (time) AS year,
  toMonth (time) AS month
FROM
  key_verifications_per_month_v3
WHERE
  outcome = 'VALID'
  AND external_id != ''
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  year,
  month;
