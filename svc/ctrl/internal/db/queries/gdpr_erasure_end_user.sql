-- name: EraseEndUserKeyPermissions :execrows
DELETE FROM keys_permissions
WHERE keys_permissions.workspace_id = sqlc.arg(workspace_id)
  AND keys_permissions.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.workspace_id = sqlc.arg(workspace_id) AND `keys`.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id)))
LIMIT 10000;

-- name: EraseEndUserKeyRoles :execrows
DELETE FROM keys_roles
WHERE keys_roles.workspace_id = sqlc.arg(workspace_id)
  AND keys_roles.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.workspace_id = sqlc.arg(workspace_id) AND `keys`.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id)))
LIMIT 10000;

-- name: EraseEndUserKeyEncryptions :execrows
DELETE FROM encrypted_keys
WHERE encrypted_keys.workspace_id = sqlc.arg(workspace_id)
  AND encrypted_keys.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.workspace_id = sqlc.arg(workspace_id) AND `keys`.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id)))
LIMIT 10000;

-- name: EraseEndUserKeyRatelimits :execrows
DELETE FROM ratelimits
WHERE ratelimits.workspace_id = sqlc.arg(workspace_id)
  AND ratelimits.key_id IN (SELECT `keys`.id FROM `keys` WHERE `keys`.workspace_id = sqlc.arg(workspace_id) AND `keys`.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id)))
LIMIT 10000;

-- name: EraseEndUserKeys :execrows
DELETE FROM `keys`
WHERE `keys`.workspace_id = sqlc.arg(workspace_id)
  AND `keys`.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id))
LIMIT 10000;

-- name: EraseEndUserIdentityRatelimits :execrows
DELETE FROM ratelimits
WHERE ratelimits.workspace_id = sqlc.arg(workspace_id)
  AND ratelimits.identity_id IN (SELECT identities.id FROM identities WHERE identities.workspace_id = sqlc.arg(workspace_id) AND identities.external_id = sqlc.arg(external_id))
LIMIT 10000;

-- name: EraseEndUserPortalSessions :execrows
DELETE FROM portal_sessions
WHERE workspace_id = sqlc.arg(workspace_id) AND external_id = sqlc.arg(external_id)
LIMIT 10000;

-- name: EraseEndUserIdentities :execrows
DELETE FROM identities
WHERE workspace_id = sqlc.arg(workspace_id) AND external_id = sqlc.arg(external_id)
LIMIT 10000;
