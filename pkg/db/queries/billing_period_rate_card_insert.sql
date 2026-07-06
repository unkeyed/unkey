-- name: InsertBillingPeriodRateCard :exec
-- First write wins: the card resolved when a period is first billed stays
-- recorded, so later card changes never re-price a closed period (R18).
INSERT IGNORE INTO billing_period_rate_cards (
    id,
    workspace_id,
    identity_id,
    year,
    month,
    rate_card_id,
    resolved_from,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(identity_id),
    sqlc.arg(year),
    sqlc.arg(month),
    sqlc.arg(rate_card_id),
    sqlc.arg(resolved_from),
    sqlc.arg(created_at)
);
