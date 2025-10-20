-- name: ListKeysByKeyAuthID :many
SELECT
  sqlc.embed(k),
  i.id as identity_id,
  i.external_id as external_id,
  i.meta as identity_meta,
  ek.encrypted as encrypted_key,
  ek.encryption_key_id as encryption_key_id

FROM `keys` k
LEFT JOIN `identities` i ON k.identity_id = i.id
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.key_auth_id = sqlc.arg(key_space_id)
AND k.id >= sqlc.arg(id_cursor)
AND (sqlc.narg(identity_id) IS NULL OR k.identity_id = sqlc.narg(identity_id))
AND k.deleted_at_m IS NULL
ORDER BY k.id ASC
LIMIT ?
;
