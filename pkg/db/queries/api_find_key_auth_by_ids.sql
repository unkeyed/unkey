-- name: FindKeyAuthsByIds :many
SELECT ka.id as key_auth_id, a.id as api_id
FROM apis a
JOIN key_auth as ka ON ka.id = a.key_auth_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
    AND a.id IN (sqlc.slice(api_ids))
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL;
