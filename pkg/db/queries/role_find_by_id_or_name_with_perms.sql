-- name: FindRoleByIdOrNameWithPerms :one
SELECT r.pk, r.id, r.workspace_id, r.project_id, r.name, r.description, r.created_at_m, r.updated_at_m, COALESCE(
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
FROM roles r
WHERE r.workspace_id = ? AND (
    r.id = sqlc.arg('search')
    OR r.name = sqlc.arg('search')
);
