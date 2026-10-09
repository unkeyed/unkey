-- name: ListRatelimitNamespaces :many
-- Newest first. The workspace_id index is stored as (workspace_id, pk), so it
-- serves both the range and the order. The cursor resumes at its row's pk, inclusive
SELECT ns.id, ns.project_id, ns.name, ns.created_at_m, ns.updated_at_m
FROM ratelimit_namespaces ns
WHERE ns.workspace_id = sqlc.arg(workspace_id)
  AND ns.deleted_at_m IS NULL
  AND (
    sqlc.arg(cursor_id) = ''
    OR ns.pk <= (
      SELECT c.pk FROM ratelimit_namespaces c
      WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
    )
  )
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(ns.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(ns.name) LIKE LOWER(sqlc.narg(search)))
ORDER BY ns.pk DESC
LIMIT ?;
