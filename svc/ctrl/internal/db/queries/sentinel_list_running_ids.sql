-- name: ListRunningSentinelIDsAndImages :many
-- ListRunningSentinelIDsAndImages returns IDs, images, and regions of all
-- running sentinels, paginated by id. Used by the rollout service to plan
-- wave assignments without fetching full sentinel rows.
SELECT id, image, region_id
FROM sentinels
WHERE desired_state = 'running'
  AND id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT ?;
