-- name: ListKeyIDsByIdentityExternalID :many
SELECT k.id
FROM `keys` k
JOIN identities i ON k.identity_id = i.id
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.deleted = false
  AND k.deleted_at_m IS NULL;
