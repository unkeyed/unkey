-- name: CountIdentityRowsByTable :one
-- Counts an end user's rows in every table a GDPR end user erasure deletes,
-- through the identity and key ids the test seeded. Used only by tests.
SELECT
  (SELECT COUNT(*) FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id)) AS identities,
  (SELECT COUNT(*) FROM portal_sessions WHERE portal_sessions.workspace_id = sqlc.arg(workspace_id) AND portal_sessions.external_id = sqlc.arg(external_id)) AS portal_sessions,
  (SELECT COUNT(*) FROM ratelimits WHERE ratelimits.identity_id = sqlc.arg(identity_id)) AS identity_ratelimits,
  (SELECT COUNT(*) FROM `keys` WHERE keys.id = sqlc.arg(key_id)) AS key_rows,
  (SELECT COUNT(*) FROM keys_permissions WHERE keys_permissions.key_id = sqlc.arg(key_id)) AS keys_permissions,
  (SELECT COUNT(*) FROM keys_roles WHERE keys_roles.key_id = sqlc.arg(key_id)) AS keys_roles,
  (SELECT COUNT(*) FROM encrypted_keys WHERE encrypted_keys.key_id = sqlc.arg(key_id)) AS encrypted_keys,
  (SELECT COUNT(*) FROM ratelimits WHERE ratelimits.key_id = sqlc.arg(ratelimit_key_id)) AS key_ratelimits;
