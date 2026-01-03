-- name: FindLiveApiByID :one
SELECT apis.*, sqlc.embed(ka)
FROM apis
JOIN key_auth as ka ON ka.id = apis.key_auth_id
WHERE apis.id = ?
    AND ka.deleted_at_m IS NULL
    AND apis.deleted_at_m IS NULL
LIMIT 1;



