-- name: FindApiByID :one
SELECT * FROM apis WHERE id = ?;

-- name: FindApiAndKeySpaceByID :one
SELECT
    a.id AS api_id,
    a.workspace_id,
    a.key_auth_id,
    a.name AS api_name,
    COALESCE(ka.id, '') AS key_space_id,
    COALESCE(ka.store_encrypted_keys, false) AS store_encrypted_keys,
    ka.default_prefix,
    ka.default_bytes
FROM apis a
LEFT JOIN key_auth ka ON ka.id = a.key_auth_id
WHERE a.id = sqlc.arg(id);
