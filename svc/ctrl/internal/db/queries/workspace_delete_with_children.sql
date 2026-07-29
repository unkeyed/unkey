-- name: DeleteWorkspacesWithChildren :exec
-- Removes the given workspaces along with everything scoped to them.
--
-- Integration tests share one MySQL container across test processes and across
-- runs, while the ctrl crons scan the whole database rather than one workspace.
-- Rows a test leaves behind are rescanned by every later run, so the seeder
-- deletes what it created once the test finishes.
DELETE w, wb, p, a, e, d
FROM workspaces w
LEFT JOIN workspace_billing wb ON wb.workspace_id = w.id
LEFT JOIN projects p ON p.workspace_id = w.id
LEFT JOIN apps a ON a.workspace_id = w.id
LEFT JOIN environments e ON e.workspace_id = w.id
LEFT JOIN deployments d ON d.workspace_id = w.id
WHERE w.id IN (sqlc.slice('ids'));
