-- name: DeleteRoleByID :exec
DELETE r, rp, kr
FROM roles r
LEFT JOIN roles_permissions rp ON rp.role_id = r.id
LEFT JOIN keys_roles kr ON kr.role_id = r.id
WHERE r.id = sqlc.arg(role_id);
