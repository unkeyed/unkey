-- name: FindKeyByID :one
SELECT
    k.*, sqlc.embed(a),
    ek.encrypted as encrypted_key,
	ek.encryption_key_id as encryption_key_id
FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
WHERE k.id = sqlc.arg(id)
AND k.deleted_at_m IS NULL
AND a.deleted_at_m IS NULL;
