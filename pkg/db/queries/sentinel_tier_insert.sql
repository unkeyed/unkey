-- name: InsertSentinelTier :exec
-- InsertSentinelTier inserts a tier row. INSERT IGNORE so repeated seed calls
-- in tests / migrations are no-ops on the unique (tier_id, version) key.
INSERT IGNORE INTO sentinel_tiers (
    id,
    tier_id,
    version,
    cpu_millicores,
    memory_mib,
    price_per_second,
    effective_from
) VALUES (
    sqlc.arg(id),
    sqlc.arg(tier_id),
    sqlc.arg(version),
    sqlc.arg(cpu_millicores),
    sqlc.arg(memory_mib),
    sqlc.arg(price_per_second),
    sqlc.arg(effective_from)
);
