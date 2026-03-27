-- Seed portal configuration for local development.
-- This creates portal-specific rows only. The customer workspace, keyspace,
-- API, and root key must be created first via the Go seed tool:
--
--     go run . dev seed local --slug awesome
--
-- That creates ws_awesome, ks_awesome, api_awesome, and a root key.
-- This script then adds portal config, branding, and a frontline route
-- on top of that workspace.
--
-- Depends on: 01-main-schema.sql (tables), go seed tool (workspace data)
-- Run manually after seeding: docker exec -i mysql mysql -u root -proot < dev/05-seed-portal.sql

USE unkey;

-- ---------------------------------------------------------------------------
-- 1. Portal configuration linked to the customer keyspace
--    Uses ks_awesome (created by `go run . dev seed local --slug awesome`)
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
  'ks_awesome',
  TRUE,
  'http://localhost:3000/portal-return',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE enabled = TRUE;

-- ---------------------------------------------------------------------------
-- 2. Portal branding
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
-- 3. Frontline route for portal
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
  'awesome.unkey.local',
  'none',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE route_type = 'portal', fully_qualified_domain_name = 'awesome.unkey.local';
