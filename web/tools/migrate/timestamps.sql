
-- keys
UPDATE keys
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE keys
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- apis
UPDATE apis
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE apis
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- key_auth

UPDATE key_auth
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE key_auth
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;


-- vercel_bindings

UPDATE vercel_bindings
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE vercel_bindings
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

UPDATE vercel_bindings
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- vercel_integrations
UPDATE vercel_integrations
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE vercel_integrations
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- workspaces
UPDATE workspaces
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE workspaces
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- ratelimit_namespaces
UPDATE ratelimit_namespaces
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE ratelimit_namespaces
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

UPDATE ratelimit_namespaces
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- ratelimit_overrides
UPDATE ratelimit_overrides
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE ratelimit_overrides
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

UPDATE ratelimit_overrides
SET deleted_at_m=UNIX_TIMESTAMP(deleted_at)*1000
WHERE deleted_at is not null;

-- permissions
UPDATE permissions
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE permissions
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

-- roles
UPDATE roles
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE roles
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

-- keys_roles
UPDATE keys_roles
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE keys_roles
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

-- roles_permissions

SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE roles_permissions
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;

-- keys_permissions
UPDATE keys_permissions
SET created_at_m=UNIX_TIMESTAMP(created_at)*1000
WHERE created_at is not null;

UPDATE keys_permissions
SET updated_at_m=UNIX_TIMESTAMP(updated_at)*1000
WHERE updated_at is not null;


--


select * from workspaces limit 100;
