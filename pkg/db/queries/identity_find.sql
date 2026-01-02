-- name: FindIdentity :one
SELECT
    i.*,
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
        FROM ratelimits rl WHERE rl.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits
FROM identities i
JOIN (
    SELECT id1.id FROM identities id1
    WHERE id1.id = sqlc.arg(identity)
      AND id1.workspace_id = sqlc.arg(workspace_id)
      AND id1.deleted = sqlc.arg(deleted)
    UNION ALL
    SELECT id2.id FROM identities id2
    WHERE id2.workspace_id = sqlc.arg(workspace_id)
      AND id2.external_id = sqlc.arg(identity)
      AND id2.deleted = sqlc.arg(deleted)
) AS identity_lookup ON i.id = identity_lookup.id
LIMIT 1;
