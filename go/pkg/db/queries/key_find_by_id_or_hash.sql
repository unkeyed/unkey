-- name: FindKeyByIdOrHash :one
SELECT
    k.*, sqlc.embed(a),
    ek.encrypted as encrypted_key,
	ek.encryption_key_id as encryption_key_id
FROM `keys` k
JOIN apis a USING(key_auth_id)
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE (CASE
    WHEN sqlc.narg(id) IS NOT NULL THEN k.id = sqlc.narg(id)
    WHEN sqlc.narg(hash) IS NOT NULL THEN k.hash = sqlc.narg(hash)
    ELSE FALSE
END) AND k.deleted_at_m IS NULL AND a.deleted_at_m IS NULL;
