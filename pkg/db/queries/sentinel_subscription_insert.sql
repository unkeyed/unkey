-- name: InsertSentinelSubscription :exec
INSERT INTO sentinel_subscriptions (
    id,
    sentinel_id,
    workspace_id,
    region_id,
    tier_id,
    tier_version,
    cpu_millicores,
    memory_mib,
    replicas,
    price_per_second,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(sentinel_id),
    sqlc.arg(workspace_id),
    sqlc.arg(region_id),
    sqlc.arg(tier_id),
    sqlc.arg(tier_version),
    sqlc.arg(cpu_millicores),
    sqlc.arg(memory_mib),
    sqlc.arg(replicas),
    sqlc.arg(price_per_second),
    sqlc.arg(created_at)
);
