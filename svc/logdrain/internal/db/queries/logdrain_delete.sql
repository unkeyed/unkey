-- DeleteLogdrain hard-deletes one drain. Only tests call it.
-- name: DeleteLogdrain :exec
DELETE FROM logdrains
WHERE id = sqlc.arg(id);
