-- name: UpdateKeySpaceKeyEncryption :exec
UPDATE `key_auth` SET store_encrypted_keys = sqlc.arg(store_encrypted_keys) WHERE id = sqlc.arg(id);
