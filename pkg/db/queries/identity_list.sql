-- name: ListIdentities :many
SELECT
    i.id,
    i.external_id,
    i.workspace_id,
    i.environment,
    i.meta,
    i.deleted,
    i.created_at,
    i.updated_at,
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', r.id,
                'name', r.name,
                'limit', r.`limit`,
                'duration', r.duration,
                'auto_apply', r.auto_apply = 1
            )
        )
        FROM ratelimits r
        WHERE r.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits
FROM identities i
WHERE i.workspace_id = sqlc.arg(workspace_id)
AND i.deleted = sqlc.arg(deleted)
AND i.id >= sqlc.arg(id_cursor)
-- search is a pre-escaped LIKE pattern built by mysql.LikeContains; NULL disables the filter
AND (sqlc.narg(search) IS NULL OR i.id LIKE sqlc.narg(search) OR i.external_id LIKE sqlc.narg(search))
ORDER BY i.id ASC
LIMIT ?
