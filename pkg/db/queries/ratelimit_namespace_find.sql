-- name: FindRatelimitNamespace :one
SELECT ns.pk, ns.id, ns.workspace_id, ns.project_id, ns.name, ns.created_at_m, ns.updated_at_m, ns.deleted_at_m,
       coalesce(
               (select json_arrayagg(
                               json_object(
                                       'id', ro.id,
                                       'identifier', ro.identifier,
                                       'limit', ro.limit,
                                       'duration', ro.duration
                               )
                       )
                from ratelimit_overrides ro where ns.id = ro.namespace_id AND ro.deleted_at_m IS NULL),
               json_array()
       ) as overrides
FROM `ratelimit_namespaces` ns
WHERE ns.workspace_id = sqlc.arg(workspace_id)
AND (ns.id = sqlc.arg(namespace) OR ns.name = sqlc.arg(namespace));
