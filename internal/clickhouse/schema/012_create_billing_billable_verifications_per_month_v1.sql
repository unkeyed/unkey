-- +goose up
CREATE TABLE billing.billable_verifications_per_month_v1
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;


-- +goose down
DROP TABLE billing.billable_verifications_per_month_v1;
