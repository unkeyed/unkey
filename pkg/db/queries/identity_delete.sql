-- name: DeleteIdentity :exec
DELETE FROM identities 
WHERE workspace_id = sqlc.arg(workspace_id) 
  AND (id = sqlc.arg(identity) OR external_id = sqlc.arg(identity));
