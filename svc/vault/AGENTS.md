# Vault

- Vault authenticates service callers, not end-user resource permissions.
  Callers must authorize access before asking Vault to decrypt. Scope storage
  and cache lookups by keyring as well as key identifier.
- Ciphertext is manually unmarshaled, so protobuf validation does not enforce
  its nonce, ciphertext, or key-ID constraints. Keep validation before decrypting
  in [rpc_decrypt.go](internal/vault/rpc_decrypt.go).
- Persist the versioned DEK object before updating `LATEST`; otherwise new
  ciphertext could name a key that was never durably stored. See
  [create_key.go](internal/keyring/create_key.go).
- In the get-or-create path, only an explicit storage not-found result permits
  key creation. A timeout, permission failure, or corrupt object is not a missing
  key. See
  [get_or_create_key.go](internal/keyring/get_or_create_key.go).
- Encryption refreshes stale `LATEST` entries, while decryption can use a cached
  immutable DEK. Do not apply the same refresh policy to both without accounting
  for rotation. `ReEncrypt` refreshes key selection before encrypting again.
- KEK rotation changes how existing DEKs are wrapped, not the DEKs used by
  ciphertext. Keep the old KEK available until objects wrapped by it have been
  migrated. See [roll_deks.go](internal/vault/roll_deks.go).
