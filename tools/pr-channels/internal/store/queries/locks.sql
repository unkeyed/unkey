-- name: InsertDelivery :execrows
-- InsertDelivery records a webhook delivery id. RowsAffected is 0 when the id
-- was already present (i.e. a duplicate delivery).
INSERT INTO webhook_deliveries (delivery_id)
VALUES (sqlc.arg(delivery_id))
ON CONFLICT DO NOTHING;

-- name: TryAcquireCronLock :execrows
-- TryAcquireCronLock takes a named lease lock when the current lease has
-- expired. RowsAffected is 1 for the winner and 0 for everyone else.
INSERT INTO cron_locks (name, holder, locked_until)
VALUES (sqlc.arg(name), sqlc.arg(holder), sqlc.arg(locked_until))
ON CONFLICT (name) DO UPDATE
    SET holder = EXCLUDED.holder, locked_until = EXCLUDED.locked_until
    WHERE cron_locks.locked_until < now();
