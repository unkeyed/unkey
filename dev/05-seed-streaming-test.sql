-- Seed data for streaming integration test
-- Sets up routing: frontline -> sentinel -> streaming-backend

USE unkey;

-- Project for streaming test
INSERT INTO projects (id, workspace_id, name, slug, created_at)
VALUES ('prj_streaming_test', 'ws_local_root', 'Streaming Test', 'streaming-test', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE name = 'Streaming Test';

-- Environment for streaming test
INSERT INTO environments (id, workspace_id, project_id, slug, sentinel_config, created_at)
VALUES ('env_streaming_test', 'ws_local_root', 'prj_streaming_test', 'production', '', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE slug = 'production';

-- Deployment for streaming test
INSERT INTO deployments (id, k8s_name, workspace_id, project_id, environment_id, image, sentinel_config, cpu_millicores, memory_mib, encrypted_environment_variables, status, created_at)
VALUES ('dep_streaming_test', 'dep-streaming-test', 'ws_local_root', 'prj_streaming_test', 'env_streaming_test', 'streaming-backend:latest', '', 100, 128, '', 'ready', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE status = 'ready';

-- Sentinel: tells frontline where to forward requests for env_streaming_test
INSERT INTO sentinels (id, workspace_id, project_id, environment_id, k8s_name, k8s_address, region, image, desired_state, health, desired_replicas, available_replicas, cpu_millicores, memory_mib, created_at)
VALUES ('stl_streaming_test', 'ws_local_root', 'prj_streaming_test', 'env_streaming_test', 'sentinel-streaming', 'sentinel:7075', 'local.dev', 'sentinel:latest', 'running', 'healthy', 1, 1, 100, 128, UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE health = 'healthy';

-- Instance: tells sentinel where to forward requests for dep_streaming_test
INSERT INTO instances (id, deployment_id, workspace_id, project_id, region, cluster_id, k8s_name, address, cpu_millicores, memory_mib, status)
VALUES ('inst_streaming_test', 'dep_streaming_test', 'ws_local_root', 'prj_streaming_test', 'local.dev', 'cluster-local', 'streaming-backend', 'streaming-backend:8085', 100, 128, 'running')
ON DUPLICATE KEY UPDATE status = 'running';

-- Frontline route: maps hostname to deployment
INSERT INTO frontline_routes (id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at)
VALUES ('route_streaming_test', 'prj_streaming_test', 'dep_streaming_test', 'env_streaming_test', 'streaming.local.dev', 'none', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE deployment_id = 'dep_streaming_test';

-- Also add localhost route for easier local testing
INSERT INTO frontline_routes (id, project_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at)
VALUES ('route_streaming_localhost', 'prj_streaming_test', 'dep_streaming_test', 'env_streaming_test', 'localhost', 'none', UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE deployment_id = 'dep_streaming_test';
