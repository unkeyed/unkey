-- name: ListIdentities :many
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
AND deleted = sqlc.arg(deleted)
AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?
