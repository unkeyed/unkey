-- name: FindIdentityWithRatelimits :many
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
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.id = sqlc.arg(identity)
  AND i.deleted = sqlc.arg(deleted)
UNION ALL
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
WHERE i.workspace_id = sqlc.arg(workspace_id)
  AND i.external_id = sqlc.arg(identity)
  AND i.deleted = sqlc.arg(deleted)
LIMIT 1;
