-- name: ListStateChanges :many
-- Returns state changes for watch loop. Includes 1-second visibility delay
-- to handle AUTO_INCREMENT gaps where sequence N+1 commits before N.
-- Clients filter by their region when fetching the actual resource.
SELECT sequence, resource_type, resource_id, op
FROM `state_changes`
WHERE region = sqlc.arg(region)
  AND sequence > sqlc.arg(after_sequence)
  AND created_at < (UNIX_TIMESTAMP() * 1000) - 1000
ORDER BY sequence ASC
LIMIT ?;
