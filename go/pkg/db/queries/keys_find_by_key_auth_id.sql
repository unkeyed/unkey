-- name: FindKeysByKeyAuthId :many
SELECT *
FROM `keys`
WHERE key_auth_id = sqlc.arg(key_auth_id)
AND deleted_at_m IS NULL;
