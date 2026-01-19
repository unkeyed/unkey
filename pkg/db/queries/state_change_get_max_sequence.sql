-- name: GetMaxStateChangeSequence :one
-- GetMaxStateChangeSequence returns the highest sequence number for a region.
-- Used during bootstrap to set the sequence boundary.
SELECT CAST(COALESCE(MAX(sequence), 0) AS UNSIGNED) AS max_sequence
FROM `state_changes`
WHERE region = sqlc.arg(region);
