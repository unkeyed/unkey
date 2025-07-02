-- name: FindKeyEncryptionByKeyID :one
SELECT * FROM encrypted_keys WHERE key_id = ?;
