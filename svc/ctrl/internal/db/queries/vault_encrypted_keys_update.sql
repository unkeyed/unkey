-- name: UpdateVaultEncryptedKeys :execrows
-- API key plaintext is immutable. The cron serializes re-encryption writes.
-- Each JSON entry is keyed by primary key and includes its workspace. UPDATE
-- preserves concurrent deletions, unlike a bulk upsert.
UPDATE encrypted_keys
SET encrypted = JSON_UNQUOTE(JSON_EXTRACT(
        CAST(sqlc.arg(updates) AS JSON), CONCAT('$."', pk, '".ciphertext')
    )),
    encryption_key_id = JSON_UNQUOTE(JSON_EXTRACT(
        CAST(sqlc.arg(updates) AS JSON), CONCAT('$."', pk, '".key_id')
    ))
WHERE pk IN (sqlc.slice(pks))
  AND BINARY workspace_id = BINARY JSON_UNQUOTE(JSON_EXTRACT(
      CAST(sqlc.arg(updates) AS JSON), CONCAT('$."', pk, '".keyring')
  ));
