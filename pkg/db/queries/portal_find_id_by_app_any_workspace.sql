-- name: FindPortalIdByAppAnyWorkspace :one
-- Reports whether any portal already claims this app, across every workspace.
--
-- Deliberately unscoped, unlike portal_find_by_app.sql: `idx_app_id` is unique
-- table-wide, so a create or update can collide with a portal the caller cannot
-- see. This answers "is the association free" for the conflict pre-check. It
-- returns only the id -- never the owning workspace -- so the caller cannot use
-- a conflict to probe another tenant.
SELECT id FROM portals
WHERE app_id = sqlc.arg('app_id')
LIMIT 1;
