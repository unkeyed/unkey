-- name: FindIdentityByID :one
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id = sqlc.arg(identity_id)
  AND deleted = sqlc.arg(deleted);

-- name: FindIdentityByExternalID :one
SELECT *
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND external_id = sqlc.arg(external_id)
  AND deleted = sqlc.arg(deleted);
