-- Credits spent per end-user identity per month, KEYED BY KEYSPACE.
-- Supersedes billable_credits_per_identity_per_month_v1 by retaining
-- key_space_id in the grain so billing can be scoped to enabled keyspaces.
-- spent_credits is recorded on the verification event regardless of outcome, so
-- no outcome filter applies here; zero-credit rows sum to zero harmlessly under
-- SummingMergeTree. Empty external_id rows are excluded (no billable subject).
CREATE TABLE billable_credits_per_identity_per_month_v2 (
  year Int16,
  month Int8,
  workspace_id String,
  key_space_id String,
  identity_id String,
  external_id String,
  spent_credits Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (workspace_id, key_space_id, external_id, identity_id, year, month);

CREATE MATERIALIZED VIEW billable_credits_per_identity_per_month_mv_v2 TO billable_credits_per_identity_per_month_v2 AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  sum(spent_credits) AS spent_credits,
  toYear (time) AS year,
  toMonth (time) AS month
FROM
  key_verifications_per_month_v3
WHERE
  external_id != ''
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  year,
  month;
