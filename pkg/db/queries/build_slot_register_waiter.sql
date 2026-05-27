-- name: RegisterBuildSlotWaiter :exec
-- Parks a deployment as a waiter for a build slot. ON DUPLICATE KEY makes
-- re-entry safe: a Deploy retry with a fresh awakeable updates the row
-- in-place so the next Release wakes the new awakeable, not a dead one.
INSERT INTO build_slot_waiters
  (deployment_id, workspace_id, awakeable_id, is_production, enqueued_at)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  awakeable_id = VALUES(awakeable_id),
  enqueued_at  = VALUES(enqueued_at);
