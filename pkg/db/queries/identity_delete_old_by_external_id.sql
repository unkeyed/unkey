-- name: DeleteOldIdentityByExternalID :exec
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
DELETE i, rl
FROM identities i
LEFT JOIN ratelimits rl ON (i.id COLLATE utf8mb4_0900_ai_ci = rl.identity_id AND i.id COLLATE utf8mb4_0900_as_cs = rl.identity_id)
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(external_id)
  AND i.id != sqlc.arg(current_identity_id)
  AND i.deleted = true;
