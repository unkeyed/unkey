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
