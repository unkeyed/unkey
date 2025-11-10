-- name: ListLiveKeysByKeySpaceID :many
SELECT k.*,
       i.id                 as identity_table_id,
       i.external_id        as identity_external_id,
       i.meta               as identity_meta,
       ek.encrypted         as encrypted_key,
       ek.encryption_key_id as encryption_key_id,
       -- Roles with both IDs and names (sorted by name)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', r.id,
                                       'name', r.name,
                                       'description', r.description
                               )
                       )
                FROM keys_roles kr
                         JOIN roles r ON r.id = kr.role_id
                WHERE kr.key_id = k.id
                ORDER BY r.name),
               JSON_ARRAY()
       )                    as roles,
       -- Direct permissions attached to the key (sorted by slug)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', p.id,
                                       'name', p.name,
                                       'slug', p.slug,
                                       'description', p.description
                               )
                       )
                FROM keys_permissions kp
                         JOIN permissions p ON kp.permission_id = p.id
                WHERE kp.key_id = k.id
                ORDER BY p.slug),
               JSON_ARRAY()
       )                    as permissions,
       -- Permissions from roles (sorted by slug)
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', p.id,
                                       'name', p.name,
                                       'slug', p.slug,
                                       'description', p.description
                               )
                       )
                FROM keys_roles kr
                         JOIN roles_permissions rp ON kr.role_id = rp.role_id
                         JOIN permissions p ON rp.permission_id = p.id
                WHERE kr.key_id = k.id
                ORDER BY p.slug),
               JSON_ARRAY()
       )                    as role_permissions,
       -- Rate limits
       COALESCE(
               (SELECT JSON_ARRAYAGG(
                               JSON_OBJECT(
                                       'id', id,
                                       'name', name,
                                       'key_id', key_id,
                                       'identity_id', identity_id,
                                       'limit', `limit`,
                                       'duration', duration,
                                       'auto_apply', auto_apply = 1
                               )
                       )
                FROM (
                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
                    FROM ratelimits rl
                    WHERE rl.key_id = k.id
                    UNION ALL
                    SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
                    FROM ratelimits rl
                    WHERE rl.identity_id = i.id
                ) AS combined_rl),
               JSON_ARRAY()
       )                    AS ratelimits
FROM `keys` k
         STRAIGHT_JOIN key_auth ka ON ka.id = k.key_auth_id
         STRAIGHT_JOIN workspaces ws ON ws.id = k.workspace_id
         LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
         LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
WHERE k.key_auth_id = sqlc.arg(key_space_id)
  AND k.id >= sqlc.arg(id_cursor)
  AND (
    sqlc.arg(external_id) = '' OR (
      i.workspace_id = k.workspace_id
      AND i.external_id = sqlc.arg(external_id)
    )
  )
  AND k.deleted_at_m IS NULL
  AND ka.deleted_at_m IS NULL
  AND ws.deleted_at_m IS NULL
ORDER BY k.id ASC
LIMIT ?;
