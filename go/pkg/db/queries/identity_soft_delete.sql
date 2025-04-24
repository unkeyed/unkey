-- name: SoftDeleteIdentity :exec
UPDATE identities set deleted = 1 WHERE id = sqlc.arg('id')
