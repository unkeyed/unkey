-- name: DeleteKeyByID :exec
DELETE k, kp, kr, rl, ek
FROM `keys` k
LEFT JOIN keys_permissions kp ON k.id = kp.key_id
LEFT JOIN keys_roles kr ON k.id = kr.key_id
LEFT JOIN ratelimits rl ON k.id = rl.key_id
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.id = ?;
