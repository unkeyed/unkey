
-- name: UpdateInstanceStatus :exec
UPDATE instances SET
	status = sqlc.arg(status)
WHERE id = sqlc.arg(id);
