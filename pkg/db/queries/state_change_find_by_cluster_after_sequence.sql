-- name: FindStateChangesByClusterAfterSequence :many
SELECT *
FROM `state_changes`
WHERE cluster_id = sqlc.arg(cluster_id)
  AND sequence > sqlc.arg(after_sequence)
ORDER BY sequence ASC;
