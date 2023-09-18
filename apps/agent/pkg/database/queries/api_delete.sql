-- name: DeleteApi :exec
DELETE FROM `apis` WHERE id = sqlc.arg("id");
