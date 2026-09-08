-- name: FindDeploymentWithEnvironmentAndApp :one
SELECT d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.app_id, d.source, d.image_requested, d.image, d.image_resolved, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.cpu_millicores, d.memory_mib, d.storage_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.port, d.shutdown_signal, d.upstream_protocol, d.healthcheck, d.pr_number, d.fork_repository_full_name, d.github_deployment_id, d.invocation_id, d.status, d.`trigger`, d.triggered_by, d.trigger_reason, d.created_at, d.updated_at, e.slug AS environment_slug, e.kind AS environment_kind, a.current_deployment_id, a.is_rolled_back
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.id = sqlc.arg(id);
