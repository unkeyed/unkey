-- name: FindIdentity :one
SELECT
    i.*,
    c.id AS credit_id,
    c.remaining AS credit_remaining,
    c.refill_amount AS credit_refill_amount,
    c.refill_day AS credit_refill_day,
    c.refilled_at AS credit_refilled_at,
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
LEFT JOIN credits c ON c.identity_id = i.id
WHERE i.id = (
    SELECT id FROM identities sub1
    WHERE sub1.workspace_id = sqlc.arg(workspace_id)
      AND sub1.id = sqlc.arg(identity)
      AND sub1.deleted = sqlc.arg(deleted)
    UNION ALL
    SELECT id FROM identities sub2
    WHERE sub2.workspace_id = sqlc.arg(workspace_id)
      AND sub2.external_id = sqlc.arg(identity)
      AND sub2.deleted = sqlc.arg(deleted)
    LIMIT 1
);
