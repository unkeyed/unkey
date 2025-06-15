-- name: UpdateIdentity :exec
UPDATE `identities` 
SET 
    meta = sqlc.arg('meta'),
    updated_at = NOW()
WHERE 
    id = sqlc.arg('id');