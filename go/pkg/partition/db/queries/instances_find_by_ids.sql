-- name: FindInstancesByIds :many
SELECT * FROM instance WHERE deployment_id IN (sqlc.slice('deployment_ids'));
