-- name: ListDesiredGateways :many
SELECT
    id as gateway_id,
    workspace_id,
    environment_id,
    k8s_service_name,
    region,
    image,
    desired_state,
    replicas,
    cpu_millicores,
    memory_mib,
    project_id
FROM `gateways`
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region))
    AND desired_state = sqlc.arg(desired_state)
    AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
