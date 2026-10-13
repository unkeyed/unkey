-- name: CountPortalsByWorkspace :one
-- CountPortalsByWorkspace counts the portals in a workspace. Intended for
-- tests that prove a rejected call wrote or deleted nothing.
SELECT COUNT(*)
FROM portals
WHERE workspace_id = sqlc.arg(workspace_id);
