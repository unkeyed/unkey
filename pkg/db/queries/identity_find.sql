-- name: FindIdentity :one
SELECT
    i.pk, i.id, i.external_id, i.workspace_id, i.project_id, i.environment, i.meta,
    i.deleted, i.created_at, i.updated_at,
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', rl.id,
                'name', rl.name,
                'key_id', rl.key_id,
                'identity_id', rl.identity_id,
                'limit', rl.`limit`,
                'duration', rl.duration,
                'auto_apply', rl.auto_apply = 1
            )
        )
        FROM ratelimits rl WHERE i.id = rl.identity_id),
        JSON_ARRAY()
    ) as ratelimits
FROM (
    SELECT id1.*, 0 AS lookup_priority FROM identities id1
    WHERE id1.id = sqlc.arg(identity)
      AND id1.workspace_id = sqlc.arg(workspace_id)
      AND id1.deleted = sqlc.arg(deleted)
    UNION ALL
    SELECT id2.*, 1 AS lookup_priority FROM identities id2
    WHERE id2.workspace_id = sqlc.arg(workspace_id)
      AND id2.external_id = sqlc.arg(identity)
      AND id2.deleted = sqlc.arg(deleted)
) AS i
ORDER BY i.lookup_priority
LIMIT 1;
