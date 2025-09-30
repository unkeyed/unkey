-- We should really have this as part of our local cli,
-- it was very helpful to quickly seed the database with
-- project data.




-- Insert test workspace
INSERT INTO workspaces (
  id,
  org_id,
  name,
  slug,
  created_at_m,
  beta_features,
  features
) VALUES (
  'ws_chronark',
  'org_chronark',
  'Chronark',
  'chronark',
  UNIX_TIMESTAMP() * 1000,
  '{"deployments":true}',
  '{}'
) ON DUPLICATE KEY UPDATE created_at_m = UNIX_TIMESTAMP() * 1000;

-- Insert project
INSERT INTO projects (
  id,
  workspace_id,
  name,
  slug,
  created_at
) VALUES (
  'proj_chronark',
  'ws_chronark',
  'API',
  'api',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at = UNIX_TIMESTAMP() * 1000;

INSERT INTO environments (
  id,
  workspace_id,
  project_id,
  slug,
  created_at
) VALUES (
  'env_chronark',
  'ws_chronark',
  'proj_chronark',
  'production',
  UNIX_TIMESTAMP() * 1000
) ON DUPLICATE KEY UPDATE created_at = UNIX_TIMESTAMP() * 1000;
