-- name: FindIdentityForBatch :one
-- transactional-batch-statement
SELECT id
FROM identities
WHERE workspace_id = sqlc.arg(workspace_id)
  AND external_id = sqlc.arg(external_id)
  AND deleted = false
LIMIT 1;
