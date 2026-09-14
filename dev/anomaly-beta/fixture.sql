START TRANSACTION;
SET @now = UNIX_TIMESTAMP() * 1000;
SET @app = (SELECT a.id FROM apps a JOIN projects p ON p.id = a.project_id
  WHERE a.workspace_id = 'ws_local' AND p.slug = 'local-api' AND a.slug = 'default');
SET @project = (SELECT project_id FROM apps WHERE id = @app);
SET @environment = (SELECT id FROM environments WHERE app_id = @app AND kind = 'production');
SET @region = (SELECT id FROM regions WHERE name = 'local' AND platform = 'dev');

INSERT INTO deployments
  (id, k8s_name, workspace_id, project_id, app_id, environment_id, sentinel_config,
   cpu_millicores, memory_mib, encrypted_environment_variables, status, created_at)
SELECT 'dep_anomaly_local_beta', 'anomaly-local-beta', 'ws_local', @project, @app,
  @environment, '{}', 250, 256, '', 'ready', @now - 86400000
WHERE NOT EXISTS (SELECT 1 FROM deployments WHERE id = 'dep_anomaly_local_beta');

UPDATE apps SET current_deployment_id = 'dep_anomaly_local_beta'
WHERE id = @app AND current_deployment_id IS NULL;

INSERT INTO deployment_topology
  (workspace_id, deployment_id, region_id, desired_status, created_at)
SELECT 'ws_local', 'dep_anomaly_local_beta', @region, 'running', @now
WHERE NOT EXISTS (SELECT 1 FROM deployment_topology
  WHERE deployment_id = 'dep_anomaly_local_beta' AND region_id = @region);

INSERT INTO clusters (id, cell_id, region_id, last_heartbeat_at)
SELECT 'cluster_anomaly_beta', 'cell_anomaly_beta', @region, @now
WHERE NOT EXISTS (SELECT 1 FROM clusters WHERE id = 'cluster_anomaly_beta');
COMMIT;
