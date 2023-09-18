-- name: DeleteKeyAuth :exec
DELETE FROM `key_auth` WHERE id = sqlc.arg("id");
