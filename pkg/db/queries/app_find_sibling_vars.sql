-- name: FindSiblingAppVarsByProjectAndEnv :many
SELECT a.slug AS app_slug, aev.`key`, aev.value
FROM app_environment_variables aev
INNER JOIN apps a ON a.id = aev.app_id
WHERE a.project_id = sqlc.arg(project_id)
  AND aev.environment_id = sqlc.arg(environment_id);
