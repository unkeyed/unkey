-- name: SentinelUpdateTier :exec
UPDATE sentinels
SET sentinel_tier_id = sqlc.arg(sentinel_tier_id),
    cpu_millicores = sqlc.arg(cpu_millicores),
    memory_mib = sqlc.arg(memory_mib),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
