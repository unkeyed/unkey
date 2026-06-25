-- name: UpdateAppRuntimeSettings :exec
-- Updates only the columns whose *_specified flag is 1, preserving all others.
-- sentinel_config is intentionally absent from the SET list so it is preserved
-- without a prior read. healthcheck and openapi_spec_path are clearable (narg).
UPDATE app_runtime_settings t
SET
    port = CASE
        WHEN CAST(sqlc.arg('port_specified') AS UNSIGNED) = 1 THEN sqlc.arg('port')
        ELSE t.port
    END,
    cpu_millicores = CASE
        WHEN CAST(sqlc.arg('cpu_millicores_specified') AS UNSIGNED) = 1 THEN sqlc.arg('cpu_millicores')
        ELSE t.cpu_millicores
    END,
    memory_mib = CASE
        WHEN CAST(sqlc.arg('memory_mib_specified') AS UNSIGNED) = 1 THEN sqlc.arg('memory_mib')
        ELSE t.memory_mib
    END,
    storage_mib = CASE
        WHEN CAST(sqlc.arg('storage_mib_specified') AS UNSIGNED) = 1 THEN sqlc.arg('storage_mib')
        ELSE t.storage_mib
    END,
    command = CASE
        WHEN CAST(sqlc.arg('command_specified') AS UNSIGNED) = 1 THEN sqlc.arg('command')
        ELSE t.command
    END,
    healthcheck = CASE
        WHEN CAST(sqlc.arg('healthcheck_specified') AS UNSIGNED) = 1 THEN sqlc.narg('healthcheck')
        ELSE t.healthcheck
    END,
    shutdown_signal = CASE
        WHEN CAST(sqlc.arg('shutdown_signal_specified') AS UNSIGNED) = 1 THEN sqlc.arg('shutdown_signal')
        ELSE t.shutdown_signal
    END,
    upstream_protocol = CASE
        WHEN CAST(sqlc.arg('upstream_protocol_specified') AS UNSIGNED) = 1 THEN sqlc.arg('upstream_protocol')
        ELSE t.upstream_protocol
    END,
    openapi_spec_path = CASE
        WHEN CAST(sqlc.arg('openapi_spec_path_specified') AS UNSIGNED) = 1 THEN sqlc.narg('openapi_spec_path')
        ELSE t.openapi_spec_path
    END,
    updated_at = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND app_id = sqlc.arg('app_id')
  AND environment_id = sqlc.arg('environment_id');
