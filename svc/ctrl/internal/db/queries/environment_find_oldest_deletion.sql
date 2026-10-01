-- name: FindOldestEnvironmentDeletion :one
SELECT CAST(COALESCE(MIN(deleting_at), 0) AS SIGNED) AS started_at
FROM environments;
