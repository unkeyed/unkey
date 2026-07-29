-- Seed the root workspace and API for local development
-- This matches what the tools/local CLI creates

USE unkey;

-- Insert root workspace
INSERT INTO workspaces (
  id,
  org_id,
  name,
  slug,
  created_at_m,
  beta_features
) VALUES (
  'ws_local_root',
  'user_REPLACE_ME',
  'Unkey',
  'unkey',
  UNIX_TIMESTAMP() * 1000,
  '{}'
) ON DUPLICATE KEY UPDATE created_at_m = UNIX_TIMESTAMP() * 1000;

INSERT INTO quota (
  workspace_id,
  requests_per_month,
  audit_logs_retention_days,
  logs_retention_days,
  team
) VALUES (
  'ws_local_root',
  150000,
  30,
  7,
  false
) ON DUPLICATE KEY UPDATE workspace_id = 'ws_local_root';

-- Insert root workspace billing row (billing state lives in workspace_billing)
INSERT INTO workspace_billing (
  workspace_id,
  tier,
  created_at_m
) VALUES (
  'ws_local_root',
  'Free',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE workspace_id = 'ws_local_root';

-- Insert the root workspace's exact default project
INSERT INTO projects (
  id,
  workspace_id,
  name,
  slug,
  delete_protection,
  created_at
) VALUES (
  'proj_local_root',
  'ws_local_root',
  'Default',
  'default',
  true,
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at = UNIX_TIMESTAMP() * 1000;

-- Insert root keyspace
INSERT INTO key_auth (
  id,
  workspace_id,
  project_id,
  created_at_m
) VALUES (
  'ks_local_root_keys',
  'ws_local_root',
  'proj_local_root',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at_m = UNIX_TIMESTAMP() * 1000;

-- Insert root API
INSERT INTO apis (
  id,
  name,
  workspace_id,
  project_id,
  auth_type,
  key_auth_id,
  created_at_m
) VALUES (
  'api_local_root_keys',
  'Unkey',
  'ws_local_root',
  'proj_local_root',
  'key',
  'ks_local_root_keys',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at_m = UNIX_TIMESTAMP() * 1000;
