-- name: SoftDeleteKeyByID :exec
UPDATE `keys` SET deleted_at_m = sqlc.arg(now) WHERE id = ?;
