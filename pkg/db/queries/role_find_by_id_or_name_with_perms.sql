-- name: FindRoleByIdOrNameWithPerms :one
-- FindRoleByIdOrNameWithPerms resolves IDs across a workspace while names
-- resolve only in the selected project because name uniqueness is project-scoped.
-- An ID match wins when search also matches a name because ORDER BY ranks IDs
-- first. For example, search "role_admin" returns the row with that ID instead
-- of a selected-project role named "role_admin".
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
  AND (
    r.id = sqlc.arg('search')
    OR (r.project_id = sqlc.arg('project_id') AND r.name = sqlc.arg('search'))
)
ORDER BY r.id = sqlc.arg('search') DESC
LIMIT 1;
