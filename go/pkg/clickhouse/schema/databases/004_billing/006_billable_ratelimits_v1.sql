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
