-- +goose up
CREATE TABLE business.active_workspaces_per_month_v1
(
  time          Date,
  workspace_id  String
)
ENGINE = SummingMergeTree()
ORDER BY (time)
;
