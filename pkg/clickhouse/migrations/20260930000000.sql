-- Root-key authentication uses an empty workspace ID so it remains separate
-- from billable API-key usage.
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
  AND workspace_id != ''
GROUP BY workspace_id, year, month;
