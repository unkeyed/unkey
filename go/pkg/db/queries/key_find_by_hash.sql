
-- name: FindKeyByHash :one
SELECT k.*,
ws.id as ws_id,
ws.enabled as ws_enabled,
ws.deleted_at_m as ws_deleted_at_m,
fws.id as for_workspace_id,
fws.enabled as for_workspace_enabled,
fws.deleted_at_m as for_workspace_deleted_at_m
FROM `keys` k
LEFT JOIN `workspaces` ws ON ws.id = k.workspace_id
LEFT JOIN `workspaces` fws ON fws.id = k.for_workspace_id
WHERE hash = sqlc.arg(hash);
