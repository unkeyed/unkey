-- name: ResolveCustomDomainEnvironments :many
-- ResolveCustomDomainEnvironments resolves the domain-list scope whenever an
-- environment identifier is supplied. It returns environment IDs after applying
-- all supplied filters to the same project/app/environment ancestry in the authorized
-- workspace. Environment ID and slug use separate index lookups, and both matches are
-- returned when an ID collides with another environment's slug.
-- Empty project or app values disable their respective filters. Missing or incompatible
-- filters return no IDs. Results are unordered and do not authorize domain access.
--
-- For example, environment=production with empty project and app values returns env_1 and
-- env_2 if both have slug production, even when they belong to different projects.
-- Adding project=payments and app=api retains only environments under matching apps
-- in matching projects. The caller then loads domains by environment_id and authorizes
-- each row without repeating the parent filters in the domain query.
SELECT e.id
FROM environments e
JOIN apps a ON a.id = e.app_id AND a.project_id = e.project_id AND a.workspace_id = e.workspace_id
JOIN projects p ON p.id = a.project_id AND p.workspace_id = e.workspace_id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.id = sqlc.arg(environment)
  AND (sqlc.arg(project) = '' OR p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (sqlc.arg(app) = '' OR a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
UNION ALL
SELECT e.id
FROM environments e
JOIN apps a ON a.id = e.app_id AND a.project_id = e.project_id AND a.workspace_id = e.workspace_id
JOIN projects p ON p.id = a.project_id AND p.workspace_id = e.workspace_id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.slug = sqlc.arg(environment)
  AND e.id <> sqlc.arg(environment)
  AND (sqlc.arg(project) = '' OR p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (sqlc.arg(app) = '' OR a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app));
