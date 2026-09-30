-- name: ResolveCustomDomainProjects :many
-- ResolveCustomDomainProjects resolves the domain-list scope when only a project
-- identifier is supplied. It returns every matching project ID in the authorized
-- workspace without enumerating apps or environments. Missing projects return no IDs.
-- Results are unordered and do not establish permission to read any domain.
--
-- For example, project=payments returns proj_1 when proj_1 has slug payments.
-- If another project in the same workspace has ID payments, both IDs are returned:
-- an ID match does not take precedence over a slug match. The caller loads domains
-- by project_id and authorizes each row.
SELECT p.id
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project));
