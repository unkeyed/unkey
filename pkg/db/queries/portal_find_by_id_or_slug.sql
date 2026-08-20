-- name: FindPortalByIdOrSlug :one
-- Resolves a portal within a workspace by either its id or its slug, matching
-- how projects/apps/environments accept a ResourceIdentifier.
--
-- UNION ALL of two index seeks instead of `id = ? OR slug = ?`, which would
-- force a scan: `portals_id_unique` and `idx_workspace_slug` each serve one arm.
SELECT p.*
FROM portals p
JOIN (
    SELECT p1.id
    FROM portals p1
    WHERE p1.id = sqlc.arg(portal) AND p1.workspace_id = sqlc.arg(workspace_id)
    UNION ALL
    SELECT p2.id
    FROM portals p2
    WHERE p2.slug = sqlc.arg(portal) AND p2.workspace_id = sqlc.arg(workspace_id)
) AS portal_lookup ON portal_lookup.id = p.id
LIMIT 1;
