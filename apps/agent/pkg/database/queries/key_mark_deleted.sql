-- name: MarkKeyDeleted :exec
UPDATE `keys`SET deleted_at = sqlc.arg("now")  WHERE id = sqlc.arg("keyID");
