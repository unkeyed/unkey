-- Minimal routing data for the frontline benchmark harness.
--
-- One route: hostname "localhost" resolves to deployment "dep_bench", which
-- has a single running instance in region bench/local pointing at the
-- harness upstream (127.0.0.1:38080). sentinel_config is '{}' so the policy
-- engine is a no-op and the measured path is pure routing + proxying.

INSERT IGNORE INTO regions (id, name, platform, can_schedule)
VALUES ('region_bench', 'bench', 'local', true);

INSERT IGNORE INTO deployments (
  id, k8s_name, workspace_id, project_id, environment_id, app_id,
  sentinel_config, cpu_millicores, memory_mib,
  encrypted_environment_variables, status, created_at
) VALUES (
  'dep_bench', 'dep-bench', 'ws_bench', 'proj_bench', 'env_bench', 'app_bench',
  '{}', 1000, 256,
  '', 'ready', UNIX_TIMESTAMP() * 1000
);

INSERT IGNORE INTO instances (
  id, deployment_id, workspace_id, project_id, app_id, region_id,
  k8s_name, address, cpu_millicores, memory_mib, status
) VALUES (
  'inst_bench', 'dep_bench', 'ws_bench', 'proj_bench', 'app_bench', 'region_bench',
  'inst-bench', '127.0.0.1:38080', 1000, 256, 'running'
);

INSERT IGNORE INTO frontline_routes (
  id, project_id, app_id, deployment_id, environment_id,
  fully_qualified_domain_name, created_at
) VALUES (
  'fr_bench', 'proj_bench', 'app_bench', 'dep_bench', 'env_bench',
  'localhost', UNIX_TIMESTAMP() * 1000
);
