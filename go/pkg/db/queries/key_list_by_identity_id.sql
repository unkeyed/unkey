-- name: ListKeysByIdentityID :many
SELECT id, hash FROM `keys`
WHERE identity_id = sqlc.arg(identity_id)
AND deleted_at_m IS NULL;
