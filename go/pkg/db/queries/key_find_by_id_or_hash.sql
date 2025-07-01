-- name: FindKeyByIdOrHash :one
SELECT
    k.*, sqlc.embed(apis)
FROM `keys` k
JOIN apis USING(key_auth_id)
WHERE CASE
    WHEN sqlc.arg(id) IS NOT NULL THEN k.id = sqlc.arg(id)
    WHEN sqlc.arg(hash) IS NOT NULL THEN k.hash = sqlc.arg(hash)
END;
