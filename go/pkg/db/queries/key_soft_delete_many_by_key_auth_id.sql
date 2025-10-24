-- name: SoftDeleteManyKeysByKeyAuthID :exec
UPDATE `keys`
SET deleted_at_m = sqlc.arg(now)
WHERE key_auth_id = sqlc.arg(key_space_id)
AND deleted_at_m IS NULL;
