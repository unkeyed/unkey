-- name: PickNextBuildSlotWaiter :one
-- Returns the next waiter for a workspace, production waiters first then
-- by enqueue order. Wrap in a tx with FOR UPDATE to serialize promotion
-- across concurrent Release calls.
SELECT deployment_id, awakeable_id
FROM build_slot_waiters
WHERE workspace_id = ?
ORDER BY is_production DESC, enqueued_at ASC
LIMIT 1
FOR UPDATE;
