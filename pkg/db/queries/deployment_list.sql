-- name: ListDeployments :many
-- Lists a workspace's deployments newest-first for the listDeployments endpoint.
-- workspace_id is always filtered by equality, so the (workspace_id, created_at,
-- id) index serves the filter, the keyset range, and the ORDER BY from a single
-- range scan with no filesort.
-- project/app/environment/status are optional filters: an empty arg disables
-- that clause via the OR short-circuit, so one query serves every combination.
-- sqlc renders an empty status set as `IN (NULL)`, which matches nothing, so
-- filter_status gates the whole status clause: empty lists every status, any
-- non-empty value restricts to the supplied set.
-- The OR-guards make project/app/environment/status non-sargable, so they apply
-- as residual predicates on top of the workspace scan rather than driving their
-- own index. Fine while per-workspace deployment counts stay bounded; if a deep
-- filter on a large workspace ever gets hot, split it into a sargable per-scope
-- variant.
-- Cursor is a deployment id: an empty cursor is the first page; otherwise we
-- look up that row's (created_at, id) and page strictly before it, matching the
-- ORDER BY so the keyset is stable across ties in created_at.
SELECT d.* FROM `deployments` d
WHERE d.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR d.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR d.app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR d.environment_id = sqlc.arg(environment_id))
  AND (sqlc.arg(filter_status) = '' OR d.status IN (sqlc.slice('statuses')))
  AND (
    sqlc.arg(cursor_id) = ''
    OR (d.created_at, d.id) < (SELECT c.created_at, c.id FROM `deployments` c WHERE c.id = sqlc.arg(cursor_id))
  )
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?;
