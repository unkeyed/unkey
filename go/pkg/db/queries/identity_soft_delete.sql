-- name: SoftDeleteIdentity :exec
UPDATE identities 
SET deleted = 1 
WHERE workspace_id = sqlc.arg('workspace_id')
 AND (id = sqlc.arg('identity') OR external_id = sqlc.arg('identity'));
