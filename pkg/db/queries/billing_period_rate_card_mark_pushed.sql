-- name: MarkBillingPeriodRateCardPushed :exec
-- Records that this identity+period was successfully pushed to the billing
-- provider. Idempotent: only stamps pushed_at the first time (COALESCE keeps
-- the earliest push timestamp), so a retry after a crash never moves it.
UPDATE billing_period_rate_cards
SET pushed_at = COALESCE(pushed_at, sqlc.arg(pushed_at)),
    updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity_id = sqlc.arg(identity_id)
  AND year = sqlc.arg(year)
  AND month = sqlc.arg(month);
