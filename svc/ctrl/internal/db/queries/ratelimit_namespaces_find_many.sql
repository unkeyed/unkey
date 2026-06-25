-- name: FindManyRatelimitNamespaces :many
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
WHERE ns.workspace_id = sqlc.arg(workspace_id)
  AND (ns.id IN (sqlc.slice(namespaces)) OR ns.name IN (sqlc.slice(namespaces)));
