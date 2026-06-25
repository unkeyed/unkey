-- name: FindKeyAuthsByKeyAuthIds :many
SELECT ka.id as key_auth_id, a.id as api_id
FROM key_auth as ka
JOIN apis a ON a.key_auth_id = ka.id
WHERE a.workspace_id = sqlc.arg(workspace_id)
    AND ka.id IN (sqlc.slice(key_auth_ids))
    AND ka.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL;
