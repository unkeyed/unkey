-- name: FindKeyByIdOrHash :one
SELECT
    k.*, sqlc.embed(apis),
    ek.encrypted as encrypted_key,
	ek.encryption_key_id as encryption_key_id
FROM `keys` k
JOIN apis USING(key_auth_id)
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE CASE
    WHEN sqlc.arg(id) IS NOT NULL THEN k.id = sqlc.arg(id)
    WHEN sqlc.arg(hash) IS NOT NULL THEN k.hash = sqlc.arg(hash)
END;
