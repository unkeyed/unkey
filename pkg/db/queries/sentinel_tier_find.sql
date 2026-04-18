-- name: FindSentinelTier :one
-- FindSentinelTier returns the tier catalog row for a (tier_id, version) pair.
-- Called by ProvisionSentinel to read the canonical tier values before
-- denormalizing them onto a new sentinel_subscriptions row.
SELECT * FROM sentinel_tiers
WHERE tier_id = sqlc.arg(tier_id) AND version = sqlc.arg(version)
LIMIT 1;
