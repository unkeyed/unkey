-- name: FindKeyAndApiByHash :one
select
    keys.id as keyId,
    keys.owner_id,
    keys.meta,
    keys.expires,
    keys.ratelimit_type,
    keys.ratelimit_limit,
    keys.ratelimit_refill_rate,
    keys.ratelimit_refill_interval,
    keys.workspace_id,
    keys.remaining_requests,
    apis.id as api_id,
    apis.ip_whitelist
from
    `keys`
    INNER JOIN `apis` ON keys.key_auth_id = apis.key_auth_id
WHERE
    keys.hash = sqlc.arg("hash");