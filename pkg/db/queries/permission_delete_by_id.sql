-- name: DeletePermission :exec
DELETE p, rp, kp
FROM permissions p
LEFT JOIN roles_permissions rp ON rp.permission_id = p.id
LEFT JOIN keys_permissions kp ON kp.permission_id = p.id
WHERE p.id = sqlc.arg(permission_id);
