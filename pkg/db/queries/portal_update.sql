-- name: UpdatePortal :execrows
-- Updates a portal's mutable fields, scoped to the workspace so one workspace can
-- never mutate another's portal.
--
-- A scoped UPDATE rather than an upsert: `portals` carries four unique keys (id,
-- workspace_id+slug, app_id, key_auth_id). ON DUPLICATE KEY UPDATE across several
-- unique indexes has undefined row selection and could silently rewrite a
-- different row, so a slug or association collision must surface as a
-- unique-constraint error the handler maps to a conflict instead.
--
-- Returns the row count so the caller can tell a real update from one that
-- matched nothing: the resolve takes no row lock, so a concurrent delete can
-- remove the row between resolving it and this statement.
--
-- Each field carries a `_specified` flag so an omitted field keeps its stored
-- value. `slug`, `display_name` and `enabled` are NOT NULL and take sqlc.arg; the two
-- associations and the two branding columns are nullable and take sqlc.narg, so
-- an explicit null clears them.
UPDATE portals p
SET
    slug = CASE
        WHEN CAST(sqlc.arg('slug_specified') AS UNSIGNED) = 1 THEN sqlc.arg('slug')
        ELSE p.slug
    END,
    display_name = CASE
        WHEN CAST(sqlc.arg('display_name_specified') AS UNSIGNED) = 1 THEN sqlc.arg('display_name')
        ELSE p.display_name
    END,
    app_id = CASE
        WHEN CAST(sqlc.arg('app_id_specified') AS UNSIGNED) = 1 THEN sqlc.narg('app_id')
        ELSE p.app_id
    END,
    key_auth_id = CASE
        WHEN CAST(sqlc.arg('key_auth_id_specified') AS UNSIGNED) = 1 THEN sqlc.narg('key_auth_id')
        ELSE p.key_auth_id
    END,
    enabled = CASE
        WHEN CAST(sqlc.arg('enabled_specified') AS UNSIGNED) = 1 THEN sqlc.arg('enabled')
        ELSE p.enabled
    END,
    logo_url = CASE
        WHEN CAST(sqlc.arg('logo_url_specified') AS UNSIGNED) = 1 THEN sqlc.narg('logo_url')
        ELSE p.logo_url
    END,
    primary_color = CASE
        WHEN CAST(sqlc.arg('primary_color_specified') AS UNSIGNED) = 1 THEN sqlc.narg('primary_color')
        ELSE p.primary_color
    END,
    updated_at = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
