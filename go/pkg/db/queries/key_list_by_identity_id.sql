-- name: ListKeysByIdentityID :many
SELECT id, hash FROM `keys` WHERE identity_id = ?;
