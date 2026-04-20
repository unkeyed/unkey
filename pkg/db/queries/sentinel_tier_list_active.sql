-- name: ListActiveSentinelTiers :many
-- ListActiveSentinelTiers returns tiers currently offered to new subscriptions.
-- A row is considered active when effective_until is NULL.
SELECT * FROM sentinel_tiers
WHERE effective_until IS NULL
ORDER BY tier_id ASC, version ASC;
