-- name: ListRolesByKeyID :many
SELECT r.*, COALESCE(
        (SELECT JSON_ARRAYAGG(
            json_object(
                'id', permission.id,
                'name', permission.name,
                'slug', permission.slug,
                'description', permission.description
           )
        )
         FROM (SELECT name, id, slug, description
               FROM roles_permissions rp
                        JOIN permissions p ON p.id = rp.permission_id
               WHERE rp.role_id = r.id) as permission),
        JSON_ARRAY()
) as permissions
FROM keys_roles kr
JOIN roles r ON kr.role_id = r.id
WHERE kr.key_id = sqlc.arg(key_id)
ORDER BY r.name;
