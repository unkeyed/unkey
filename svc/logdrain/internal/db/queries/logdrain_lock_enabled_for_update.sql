-- LockEnabledLogdrainsForUpdate locks enabled config rows before lease state.
-- SKIP LOCKED lets another lease manager or a dashboard transaction proceed
-- without creating a config-to-state versus state-to-config deadlock.
-- name: LockEnabledLogdrainsForUpdate :many
SELECT id
FROM logdrains
WHERE id IN (sqlc.slice('logdrain_ids'))
  AND enabled = true
ORDER BY id
FOR UPDATE SKIP LOCKED;
