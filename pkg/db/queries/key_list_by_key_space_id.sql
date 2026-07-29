-- name: ListKeysByKeySpaceID :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
  sqlc.embed(k),
  i.id as identity_id,
  i.external_id as external_id,
  i.meta as identity_meta,
  ek.encrypted as encrypted_key,
  ek.encryption_key_id as encryption_key_id

FROM `keys` k
LEFT JOIN `identities` i ON (k.identity_id COLLATE utf8mb4_0900_ai_ci = i.id AND k.identity_id COLLATE utf8mb4_0900_as_cs = i.id)
LEFT JOIN encrypted_keys ek ON (k.id COLLATE utf8mb4_0900_ai_ci = ek.key_id AND k.id COLLATE utf8mb4_0900_as_cs = ek.key_id)
WHERE k.key_auth_id = sqlc.arg(key_space_id)
AND k.id >= sqlc.arg(id_cursor)
AND (sqlc.narg(identity_id) IS NULL OR k.identity_id = sqlc.narg(identity_id))
AND k.deleted_at_m IS NULL
ORDER BY k.id ASC
LIMIT ?
;
