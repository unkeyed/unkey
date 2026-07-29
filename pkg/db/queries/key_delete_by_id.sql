-- name: DeleteKeyByID :exec
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
DELETE k, kp, kr, rl, ek
FROM `keys` k
LEFT JOIN keys_permissions kp ON (k.id COLLATE utf8mb4_0900_ai_ci = kp.key_id AND k.id COLLATE utf8mb4_0900_as_cs = kp.key_id)
LEFT JOIN keys_roles kr ON (k.id COLLATE utf8mb4_0900_ai_ci = kr.key_id AND k.id COLLATE utf8mb4_0900_as_cs = kr.key_id)
LEFT JOIN ratelimits rl ON (k.id COLLATE utf8mb4_0900_ai_ci = rl.key_id AND k.id COLLATE utf8mb4_0900_as_cs = rl.key_id)
LEFT JOIN encrypted_keys ek ON (k.id COLLATE utf8mb4_0900_ai_ci = ek.key_id AND k.id COLLATE utf8mb4_0900_as_cs = ek.key_id)
WHERE k.id = ?;
