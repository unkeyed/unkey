-- name: DeleteIdentity :exec
DELETE FROM identities WHERE id = sqlc.arg('id')
