-- name: InsertRateCard :exec
INSERT INTO rate_cards (
    id,
    workspace_id,
    name,
    currency,
    config,
    selectable,
    archived,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(name),
    sqlc.arg(currency),
    sqlc.arg(config),
    sqlc.arg(selectable),
    false,
    sqlc.arg(created_at)
);
