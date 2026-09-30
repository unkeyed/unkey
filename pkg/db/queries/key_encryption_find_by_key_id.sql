-- name: FindKeyEncryptionByKeyID :one
SELECT encrypted_keys.pk, encrypted_keys.workspace_id, encrypted_keys.key_id, encrypted_keys.created_at, encrypted_keys.updated_at, encrypted_keys.encrypted, encrypted_keys.encryption_key_id FROM encrypted_keys WHERE key_id = ?;
