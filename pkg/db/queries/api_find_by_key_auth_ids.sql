-- name: FindApisByKeyAuthIds :many
-- Maps keyspace ids back to the api that owns them, scoped to a workspace.
-- apis.key_auth_id is unique, so each keyspace resolves to at most one api and project.
SELECT ka.id as key_auth_id, ka.project_id, a.id as api_id
FROM apis a
JOIN key_auth as ka ON ka.id = a.key_auth_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
    AND ka.id IN (sqlc.slice(key_auth_ids))
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL;
