-- Seed a sample customer workspace with portal configuration for local development.
-- This represents a real Unkey customer ("Awesome Corp") who has enabled the
-- Customer Portal for their end users. Separate from ws_local_root, which is
-- Unkey's own internal workspace.
--
-- Depends on: 01-main-schema.sql (tables must exist)

USE unkey;

-- ---------------------------------------------------------------------------
-- 1. Customer workspace
-- ---------------------------------------------------------------------------
INSERT INTO workspaces (
  id,
  org_id,
  name,
  slug,
  created_at_m,
  beta_features,
  features
) VALUES (
  'ws_awesome',
  'org_awesome_local',
  'Awesome Corp',
  'awesome',
  UNIX_TIMESTAMP() * 1000,
  '{}',
  '{}'
) ON DUPLICATE KEY UPDATE name = 'Awesome Corp';

INSERT INTO quota (
  workspace_id,
  requests_per_month,
  audit_logs_retention_days,
  logs_retention_days,
  team
) VALUES (
  'ws_awesome',
  150000,
  30,
  7,
  false
) ON DUPLICATE KEY UPDATE workspace_id = 'ws_awesome';

-- ---------------------------------------------------------------------------
-- 2. Customer keyspace + API
-- ---------------------------------------------------------------------------
INSERT INTO key_auth (
  id,
  workspace_id,
  created_at_m
) VALUES (
  'ks_awesome_keys',
  'ws_awesome',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at_m = UNIX_TIMESTAMP() * 1000;

INSERT INTO apis (
  id,
  name,
  workspace_id,
  auth_type,
  key_auth_id,
  created_at_m
) VALUES (
  'api_awesome',
  'Awesome API',
  'ws_awesome',
  'key',
  'ks_awesome_keys',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE name = 'Awesome API';

-- ---------------------------------------------------------------------------
-- 3. Root key for the customer workspace (used to call portal.createSession)
--    Plain-text value for local testing: awesome_root_key_secret
-- ---------------------------------------------------------------------------
INSERT INTO `keys` (
  id,
  key_auth_id,
  hash,
  start,
  workspace_id,
  for_workspace_id,
  created_at_m
) VALUES (
  'key_awesome_root',
  'ks_local_root_keys',
  SHA2('awesome_root_key_secret', 256),
  'awesome',
  'ws_local_root',
  'ws_awesome',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE hash = SHA2('awesome_root_key_secret', 256);

-- ---------------------------------------------------------------------------
-- 4. Portal configuration
-- ---------------------------------------------------------------------------
INSERT INTO portal_configurations (
  id,
  workspace_id,
  key_auth_id,
  enabled,
  return_url,
  created_at
) VALUES (
  'portal_awesome',
  'ws_awesome',
  'ks_awesome_keys',
  TRUE,
  'http://localhost:3000/portal-return',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE enabled = TRUE;

-- ---------------------------------------------------------------------------
-- 5. Portal branding
-- ---------------------------------------------------------------------------
INSERT INTO portal_branding (
  portal_config_id,
  logo_url,
  primary_color,
  secondary_color,
  created_at
) VALUES (
  'portal_awesome',
  'https://avatars.githubusercontent.com/u/138932600',
  '#2563eb',
  '#f0f9ff',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE primary_color = '#2563eb';

-- ---------------------------------------------------------------------------
-- 6. Frontline route for portal
-- ---------------------------------------------------------------------------
INSERT INTO frontline_routes (
  id,
  route_type,
  portal_config_id,
  path_prefix,
  fully_qualified_domain_name,
  sticky,
  created_at
) VALUES (
  'fr_portal_awesome',
  'portal',
  'portal_awesome',
  '/portal',
  'awesome.localhost',
  'none',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE route_type = 'portal';
