-- name: InsertKeyEncryption :exec
-- transactional-batch-statement
INSERT INTO encrypted_keys
(workspace_id, key_id, encrypted, encryption_key_id, created_at)
VALUES (?, ?, ?, ?, ?);
