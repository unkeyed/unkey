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
