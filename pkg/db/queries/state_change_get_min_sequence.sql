-- name: GetMinStateChangeSequence :one
-- Returns the lowest retained sequence for a region.
-- Used to detect if a client's watermark is too old (requires full resync).
SELECT CAST(COALESCE(MIN(sequence), 0) AS UNSIGNED) AS min_sequence
FROM `state_changes`
WHERE region = sqlc.arg(region);
