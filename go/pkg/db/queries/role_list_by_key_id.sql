-- name: ListRolesByKeyID :many
SELECT r.*
FROM roles r
JOIN keys_roles kr ON r.id = kr.role_id
WHERE kr.key_id = sqlc.arg(key_id)
ORDER BY r.name;
