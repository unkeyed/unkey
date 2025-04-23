-- name: FindIdentityByID :one
SELECT * FROM identities WHERE id = sqlc.arg(id) AND deleted = sqlc.arg(deleted);
