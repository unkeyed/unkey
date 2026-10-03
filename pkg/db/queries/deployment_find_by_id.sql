-- name: FindDeploymentById :one
SELECT pk, id, k8s_name, workspace_id, project_id, environment_id, app_id,
    source, image_requested, image_resolved, build_id, git_commit_sha, git_branch,
    git_commit_message, git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp,
    sentinel_config, cpu_millicores, memory_mib, storage_mib, desired_state,
    encrypted_environment_variables, command, port, shutdown_signal, upstream_protocol, healthcheck,
    pr_number, fork_repository_full_name, github_deployment_id, invocation_id, status,
    first_ready_at, `trigger`, triggered_by, trigger_reason, created_at, updated_at
FROM deployments WHERE id = sqlc.arg(id);
