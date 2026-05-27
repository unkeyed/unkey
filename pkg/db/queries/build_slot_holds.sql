-- name: HoldsBuildSlot :one
-- Returns true when this deployment already owns a row in build_slots
-- (acquired directly, or pre-granted by a previous Release). Used at the
-- top of waitForBuildSlot's loop as the source of truth: any path that
-- ends with our row in build_slots means we own the slot, regardless of
-- whether the awakeable wake-up landed.
SELECT EXISTS(
    SELECT 1 FROM build_slots WHERE deployment_id = sqlc.arg('deployment_id')
) AS holds;
