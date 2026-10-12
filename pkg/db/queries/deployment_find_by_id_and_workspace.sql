-- name: FindDeploymentByIdAndWorkspace :one
-- Selects the same columns as ListDeployments, so a row converts to
-- ListDeploymentsRow for the shared response mapper
SELECT d.id, d.project_id, d.app_id, d.environment_id, d.source, d.image_requested, d.image_resolved, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.cpu_millicores, d.memory_mib, d.storage_mib, d.desired_state, d.command, d.port, d.shutdown_signal, d.upstream_protocol, d.healthcheck, d.pr_number, d.fork_repository_full_name, d.status, d.`trigger`, d.triggered_by, d.created_at, d.updated_at FROM `deployments` d
WHERE d.id = sqlc.arg(id) AND d.workspace_id = sqlc.arg(workspace_id);
