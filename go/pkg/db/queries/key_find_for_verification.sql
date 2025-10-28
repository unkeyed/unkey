-- name: FindKeyForVerification :one
select k.id,
       k.key_auth_id,
       k.workspace_id,
       k.for_workspace_id,
       k.name,
       k.meta,
       k.expires,
       k.deleted_at_m,
       k.refill_day,
       k.refill_amount,
       k.last_refill_at,
       k.enabled,
       k.remaining_requests,
       a.ip_whitelist,
       a.workspace_id  as api_workspace_id,
       a.id            as api_id,
       a.deleted_at_m  as api_deleted_at_m,

       COALESCE(
               (SELECT JSON_ARRAYAGG(name)
                FROM (SELECT name
                      FROM keys_roles kr
                               JOIN roles r ON r.id = kr.role_id
                      WHERE kr.key_id = k.id) as roles),
               JSON_ARRAY()
       )               as roles,

       COALESCE(
               (SELECT JSON_ARRAYAGG(slug)
                FROM (SELECT slug
                      FROM keys_permissions kp
                               JOIN permissions p ON kp.permission_id = p.id
                      WHERE kp.key_id = k.id

                      UNION ALL

                      SELECT slug
                      FROM keys_roles kr
                               JOIN roles_permissions rp ON kr.role_id = rp.role_id
                               JOIN permissions p ON rp.permission_id = p.id
                      WHERE kr.key_id = k.id) as combined_perms),
               JSON_ARRAY()
       )               as permissions,

       coalesce(
               (select json_arrayagg(
                    json_object(
                       'id', rl.id,
                       'name', rl.name,
                       'key_id', rl.key_id,
                       'identity_id', rl.identity_id,
                       'limit', rl.limit,
                       'duration', rl.duration,
                       'auto_apply', rl.auto_apply
                    )
                )
                from `ratelimits` rl
                where rl.key_id = k.id
                   OR rl.identity_id = i.id),
               json_array()
       ) as ratelimits,

       i.id as identity_id,
       i.external_id,
       i.meta          as identity_meta,
       ka.deleted_at_m as key_auth_deleted_at_m,
       ws.enabled      as workspace_enabled,
       fws.enabled     as for_workspace_enabled,

       -- Key-level credits
       key_credits.id as key_credit_id,
       key_credits.remaining as key_credit_remaining,
       key_credits.refill_amount as key_credit_refill_amount,
       key_credits.refill_day as key_credit_refill_day,
       key_credits.refilled_at as key_credit_refilled_at,

       -- Identity-level credits
       identity_credits.id as identity_credit_id,
       identity_credits.remaining as identity_credit_remaining,
       identity_credits.refill_amount as identity_credit_refill_amount,
       identity_credits.refill_day as identity_credit_refill_day,
       identity_credits.refilled_at as identity_credit_refilled_at

from `keys` k
         JOIN apis a USING (key_auth_id)
         JOIN key_auth ka ON ka.id = k.key_auth_id
         JOIN workspaces ws ON ws.id = k.workspace_id
         LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = 0
         LEFT JOIN credits key_credits ON key_credits.key_id = k.id
         LEFT JOIN credits identity_credits ON identity_credits.identity_id = i.id
where k.hash = ?
  and k.deleted_at_m is null;
