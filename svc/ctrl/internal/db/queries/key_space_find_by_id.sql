-- name: FindKeySpaceByID :one
SELECT * FROM `key_auth` WHERE id = sqlc.arg(id);
