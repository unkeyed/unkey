-- name: FindIdentityByExternalID :one
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND external_id = sqlc.arg(external_id)
  AND deleted = sqlc.arg(deleted);
