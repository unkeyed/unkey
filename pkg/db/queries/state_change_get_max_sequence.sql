-- name: GetMaxStateChangeSequence :one
-- Returns the highest sequence for a region.
-- Used during bootstrap to get the watermark before streaming current state.
SELECT CAST(COALESCE(MAX(sequence), 0) AS UNSIGNED) AS max_sequence
FROM `state_changes`
WHERE region = sqlc.arg(region);
