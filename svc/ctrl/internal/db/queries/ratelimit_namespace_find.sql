-- name: FindRatelimitNamespace :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT *,
       coalesce(
               (select json_arrayagg(
                               json_object(
                                       'id', ro.id,
                                       'identifier', ro.identifier,
                                       'limit', ro.limit,
                                       'duration', ro.duration
                               )
                       )
                from ratelimit_overrides ro where (ro.namespace_id = ns.id COLLATE utf8mb4_0900_ai_ci AND ro.namespace_id = ns.id COLLATE utf8mb4_0900_as_cs) AND ro.deleted_at_m IS NULL),
               json_array()
       ) as overrides
FROM `ratelimit_namespaces` ns
WHERE ns.workspace_id = sqlc.arg(workspace_id)
AND (ns.id = sqlc.arg(namespace) OR ns.name = sqlc.arg(namespace));
