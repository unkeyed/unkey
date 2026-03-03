-- name: FindProjectWithEnvironmentSettingsAndVars :one
SELECT
    sqlc.embed(p),
    sqlc.embed(e),
    sqlc.embed(ebs),
    sqlc.embed(ers),
    COALESCE(
        (SELECT JSON_ARRAYAGG(JSON_OBJECT('key', ev.`key`, 'value', ev.value))
         FROM environment_variables ev
         WHERE ev.environment_id = e.id),
        JSON_ARRAY()
    ) AS environment_variables
FROM projects p
INNER JOIN environments e
    ON e.project_id = p.id AND e.workspace_id = p.workspace_id
INNER JOIN environment_build_settings ebs
    ON ebs.environment_id = e.id
INNER JOIN environment_runtime_settings ers
    ON ers.environment_id = e.id
WHERE p.id = sqlc.arg(project_id)
  AND e.slug = sqlc.arg(slug);
