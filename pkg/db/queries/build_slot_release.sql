-- name: ReleaseBuildSlot :exec
DELETE FROM build_slots WHERE deployment_id = ?;
