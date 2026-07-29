-- name: FindManyRolesByNamesWithPerms :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
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
                        JOIN permissions p ON (p.id = rp.permission_id COLLATE utf8mb4_0900_ai_ci AND p.id = rp.permission_id COLLATE utf8mb4_0900_as_cs)
               WHERE (rp.role_id = r.id COLLATE utf8mb4_0900_ai_ci AND rp.role_id = r.id COLLATE utf8mb4_0900_as_cs)) as permission),
        JSON_ARRAY()
) as permissions
FROM roles r
WHERE r.workspace_id = ? AND r.name IN (sqlc.slice('names'));
