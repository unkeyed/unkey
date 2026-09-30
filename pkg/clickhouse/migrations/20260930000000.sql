-- Root-key authentication is logged under the workspace that owns the key,
-- but it is not billable API-key usage. New root keys have no keyspace.
ALTER TABLE `default`.`billable_verifications_per_month_mv_v2` MODIFY QUERY
SELECT
  workspace_id,
  sum(count) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM default.key_verifications_per_month_v3
WHERE outcome = 'VALID'
  AND source != 'gateway'
  AND key_space_id != ''
GROUP BY workspace_id, year, month;
