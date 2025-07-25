-- name: FindIdentity :one
SELECT * 
FROM identities 
WHERE workspace_id = sqlc.arg(workspace_id) 
 AND (external_id = sqlc.arg(identity) OR id = sqlc.arg(identity)) 
 AND deleted = sqlc.arg(deleted);
