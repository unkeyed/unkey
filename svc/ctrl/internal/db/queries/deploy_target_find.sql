-- name: FindDeployTarget :one
SELECT
    p.workspace_id AS workspace_id,
    w.slug AS workspace_slug,
    p.id AS project_id,
    a.id AS app_id,
    a.source_type AS source_type,
    a.default_branch AS default_branch,
    a.current_deployment_id AS current_deployment_id,
    e.id AS environment_id,
    e.slug AS environment_slug,
    abs.dockerfile AS dockerfile,
    abs.docker_context AS docker_context,
    abs.build_command AS build_command,
    ars.port AS port,
    ars.cpu_millicores AS cpu_millicores,
    ars.memory_mib AS memory_mib,
    ars.storage_mib AS storage_mib,
    ars.command AS command,
    ars.healthcheck AS healthcheck,
    ars.shutdown_signal AS shutdown_signal,
    ars.upstream_protocol AS upstream_protocol,
    ars.sentinel_config AS sentinel_config,
    grc.installation_id AS github_installation_id,
    grc.repository_full_name AS github_repository_full_name,
    grc.default_branch AS github_default_branch,
    aso.image_reference AS oci_image_reference,
    abs.app_id IS NOT NULL AS has_build_settings,
    b.plan AS plan,
    b.plan_override AS plan_override,
    b.spend_suspended AS spend_suspended,
    EXISTS (
        SELECT 1
        FROM app_regional_settings ars2
        INNER JOIN regions r ON r.id = ars2.region_id
        WHERE ars2.app_id = a.id
          AND ars2.environment_id = e.id
          AND r.can_schedule
    ) AS has_schedulable_region
FROM apps a
INNER JOIN projects p ON p.id = a.project_id
INNER JOIN workspaces w ON w.id = p.workspace_id
INNER JOIN environments e ON e.app_id = a.id AND e.project_id = a.project_id
INNER JOIN (
    SELECT e1.id
    FROM environments e1
    WHERE e1.app_id = sqlc.arg(app_id) AND e1.id = sqlc.arg(environment)
    UNION ALL
    SELECT e2.id
    FROM environments e2
    WHERE e2.app_id = sqlc.arg(app_id) AND e2.slug = sqlc.arg(environment)
) AS env_lookup ON env_lookup.id = e.id
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = e.id
LEFT JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = e.id
LEFT JOIN github_repo_connections grc ON grc.app_id = a.id
LEFT JOIN app_source_oci aso ON aso.app_id = a.id
LEFT JOIN workspace_billing b ON b.workspace_id = p.workspace_id
WHERE a.id = sqlc.arg(app_id)
  AND a.project_id = sqlc.arg(project_id)
LIMIT 1;
