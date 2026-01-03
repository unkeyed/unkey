-- name: FindIdentities :many
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
 AND deleted = sqlc.arg(deleted)
 AND (external_id IN(sqlc.slice(identities)) OR id IN (sqlc.slice(identities)));
