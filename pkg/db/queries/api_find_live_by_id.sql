-- name: FindLiveApiByID :one
SELECT
    apis.id AS api_id,
    apis.name AS api_name,
    apis.workspace_id AS api_workspace_id,
    apis.key_auth_id AS api_key_auth_id,
    ka.id AS key_auth_id,
    ka.project_id AS key_auth_project_id,
    ka.store_encrypted_keys AS key_auth_store_encrypted_keys
FROM apis
JOIN key_auth as ka ON ka.id = apis.key_auth_id
WHERE apis.id = ?
    AND ka.deleted_at_m IS NULL
    AND apis.deleted_at_m IS NULL
LIMIT 1;

