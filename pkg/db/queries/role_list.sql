-- name: ListRoles :many
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
WHERE r.workspace_id = sqlc.arg(workspace_id)
AND r.id >= sqlc.arg(id_cursor)
-- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
AND (sqlc.narg(search) IS NULL OR LOWER(r.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(r.name) LIKE LOWER(sqlc.narg(search)) OR LOWER(r.description) LIKE LOWER(sqlc.narg(search)))
ORDER BY r.id
LIMIT ?;
