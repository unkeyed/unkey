-- name: ListDeployments :many
-- has_status_filter and has_branch_filter gate their clauses; without them sqlc
-- renders an empty set as IN (NULL), which matches nothing.
-- Rows come newest first by created_at, the time the dashboard sorts and filters
-- by. pk is insertion order and can disagree with created_at, which would make
-- pages under a time filter skip or repeat rows, so pk only breaks ties within a
-- millisecond. The cursor names a deployment and resumes at its
-- (created_at, pk), inclusive. MySQL cannot range-scan a row comparison, so the
-- separate created_at bound lets the (app|project|workspace, created_at) index
-- seek to the cursor instead of walking every newer row
SELECT d.id, d.source, d.image_requested, d.image_resolved, d.git_commit_sha, d.git_branch, d.git_commit_message, d.git_commit_author_handle, d.git_commit_author_avatar_url, d.git_commit_timestamp, d.cpu_millicores, d.memory_mib, d.storage_mib, d.desired_state, d.command, d.port, d.shutdown_signal, d.upstream_protocol, d.healthcheck, d.pr_number, d.fork_repository_full_name, d.status, d.`trigger`, d.triggered_by, d.created_at, d.updated_at FROM `deployments` d
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR d.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR d.app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR d.environment_id = sqlc.arg(environment_id))
  AND (sqlc.arg(has_status_filter) = FALSE OR d.status IN (sqlc.slice('statuses')))
  AND (sqlc.arg(has_branch_filter) = FALSE OR d.git_branch IN (sqlc.slice('branches')))
  AND (sqlc.narg(start_time) IS NULL OR d.created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time) IS NULL OR d.created_at < sqlc.narg(end_time))
  AND (
    sqlc.arg(cursor_id) = ''
    OR (
      d.created_at <= (
        SELECT c.created_at FROM `deployments` c
        WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
      )
      AND (d.created_at, d.pk) <= (
        SELECT c.created_at, c.pk FROM `deployments` c
        WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
      )
    )
  )
ORDER BY d.created_at DESC, d.pk DESC
LIMIT ?;
