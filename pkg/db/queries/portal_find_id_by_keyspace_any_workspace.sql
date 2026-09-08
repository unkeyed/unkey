-- name: FindPortalIdByKeyspaceAnyWorkspace :one
-- Reports whether any portal already claims this keyspace, across every
-- workspace. See portal_find_id_by_app_any_workspace.sql for why this is
-- unscoped and why it returns only the id.
SELECT id FROM portals
WHERE key_auth_id = sqlc.arg('key_auth_id')
LIMIT 1;
