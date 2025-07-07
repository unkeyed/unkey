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
       a.id as api_id,
       a.deleted_at_m as api_deleted_at_m,

       COALESCE(
          (SELECT JSON_ARRAYAGG(name)
                 FROM (SELECT name
                       FROM keys_roles kr
                                JOIN roles r ON r.id = kr.role_id
                       WHERE kr.key_id = k.id) as roles),
                JSON_ARRAY()
       ) as roles,

       COALESCE((SELECT JSON_ARRAYAGG(slug)
                 FROM (SELECT slug
                       FROM keys_permissions kp
                                JOIN permissions p ON kp.permission_id = p.id
                       WHERE kp.key_id = k.id) as direct_perms
                 UNION ALL
                 SELECT slug
                 FROM (SELECT slug
                       FROM keys_roles kr
                                JOIN roles_permissions rp ON kr.role_id = rp.role_id
                                JOIN permissions p ON rp.permission_id = p.id
                       WHERE kr.key_id = k.id) as role_permissions),
                JSON_ARRAY()
       ) as perms,

       coalesce(
               (select json_arrayagg(json_array(rl.id, rl.name, rl.key_id, rl.identity_id, rl.limit, rl.duration, rl.auto_apply))
                from `ratelimits` rl
                where rl.key_id = k.id
                   OR rl.identity_id = i.id),
               json_array()
       ) as `ratelimits`,

       i.external_id,
       i.meta as identity_meta,
       sqlc.embed(ka),
       ws.enabled as workspace_enabled,
       fws.enabled as for_workspace_enabled
from `keys` k
        JOIN apis a USING(key_auth_id)
        LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = 0
        JOIN key_auth ka ON ka.id = k.key_auth_id
        JOIN workspaces ws ON ws.id = k.workspace_id
        LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
where k.hash = ?
  and k.deleted_at_m is null;
