-- name: FindBillingPeriodRateCard :one
SELECT *
FROM billing_period_rate_cards
WHERE workspace_id = sqlc.arg(workspace_id)
  AND identity_id = sqlc.arg(identity_id)
  AND year = sqlc.arg(year)
  AND month = sqlc.arg(month);
