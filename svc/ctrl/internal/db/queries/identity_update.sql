-- name: UpdateIdentity :exec
UPDATE `identities`
SET
    meta = CAST(sqlc.arg('meta') AS JSON),
    updated_at = NOW()
WHERE
    id = sqlc.arg('id');
