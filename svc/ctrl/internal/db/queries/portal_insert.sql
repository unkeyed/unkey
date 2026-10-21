-- name: InsertPortal :exec
-- Creates a bare portal row. Portals are created through the API, so ctrl only
-- needs this to seed tests that attach domains to one.
INSERT INTO portals (id, workspace_id, project_id, slug, display_name, created_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(slug),
    sqlc.arg(display_name),
    sqlc.arg(created_at)
);
