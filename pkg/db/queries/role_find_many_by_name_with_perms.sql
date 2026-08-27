-- name: FindManyRolesByNamesWithPerms :many
-- FindManyRolesByNamesWithPerms returns the requested roles and their
-- permissions from one project. The project filter prevents cross-project key
-- assignments.
SELECT *, COALESCE(
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
WHERE r.workspace_id = sqlc.arg('workspace_id')
  AND r.project_id = sqlc.arg('project_id')
  AND r.name IN (sqlc.slice('names'));
