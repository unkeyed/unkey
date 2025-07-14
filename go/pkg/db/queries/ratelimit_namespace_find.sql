-- name: FindRatelimitNamespace :one
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
                from ratelimit_overrides ro where ro.namespace_id = ns.id AND ro.deleted_at_m IS NULL),
               json_array()
       ) as overrides
FROM `ratelimit_namespaces` ns
WHERE ns.workspace_id = ?
AND CASE WHEN sqlc.narg('name') IS NOT NULL THEN ns.name = sqlc.narg('name')
WHEN sqlc.narg('id') IS NOT NULL THEN ns.id = sqlc.narg('id')
ELSE false END;
