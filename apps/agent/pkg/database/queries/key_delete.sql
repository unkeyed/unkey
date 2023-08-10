-- name: DeleteKey :exec
DELETE FROM `keys` WHERE id = sqlc.arg("id");
