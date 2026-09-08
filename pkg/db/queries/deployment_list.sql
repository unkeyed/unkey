-- name: ListDeployments :many
-- has_status_filter gates the status clause; without it sqlc renders an empty
-- status set as IN (NULL), which matches nothing.
SELECT d.pk, d.id, d.k8s_name, d.workspace_id, d.project_id, d.environment_id, d.app_id, d.source, d.image_requested, d.image, d.image_resolved, d.build_id, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.sentinel_config, d.cpu_millicores, d.memory_mib, d.storage_mib, d.desired_state, d.encrypted_environment_variables, d.command, d.port, d.shutdown_signal, d.upstream_protocol, d.healthcheck, d.pr_number, d.fork_repository_full_name, d.github_deployment_id, d.invocation_id, d.status, d.`trigger`, d.triggered_by, d.trigger_reason, d.created_at, d.updated_at FROM `deployments` d
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR d.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR d.app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR d.environment_id = sqlc.arg(environment_id))
  AND (sqlc.arg(has_status_filter) = FALSE OR d.status IN (sqlc.slice('statuses')))
  AND (
    sqlc.arg(cursor_id) = ''
    OR d.pk <= (SELECT c.pk FROM `deployments` c WHERE c.id = sqlc.arg(cursor_id))
  )
ORDER BY d.pk DESC
LIMIT ?;
