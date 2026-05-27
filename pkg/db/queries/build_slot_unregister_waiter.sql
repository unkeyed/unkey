-- name: UnregisterBuildSlotWaiter :exec
DELETE FROM build_slot_waiters WHERE deployment_id = ?;
