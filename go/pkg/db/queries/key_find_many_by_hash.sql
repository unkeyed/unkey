-- name: FindKeysByHash :many
SELECT id, hash FROM `keys` WHERE hash IN (sqlc.slice(hashes));
