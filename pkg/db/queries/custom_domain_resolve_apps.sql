-- name: ResolveCustomDomainApps :many
-- ResolveCustomDomainApps resolves the domain-list scope when app is supplied and
-- environment is omitted. It returns app IDs, not the IDs of their environments,
-- so the domain query can select every domain under those apps without parent joins.
--
-- Both app and project match ID or slug, with no preference for ID matches. An empty
-- project disables that filter. When supplied, project must match the app's actual
-- parent in the authorized workspace. Missing or incompatible filters return no IDs.
-- Results are unordered and do not establish permission to read any domain.
--
-- For example, app=api and project=payments returns app_1 when app_1 has slug api
-- under the project with slug payments. If app_2 also has slug api under billing,
-- it is excluded. With an empty project, both app_1 and app_2 are returned. Neither case
-- enumerates environments; the caller loads domains by app_id and authorizes each row.
SELECT a.id
FROM apps a
JOIN projects p ON p.id = a.project_id AND p.workspace_id = a.workspace_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
  AND (sqlc.arg(project) = '' OR p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project));
