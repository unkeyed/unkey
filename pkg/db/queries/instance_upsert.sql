
-- name: UpsertInstance :exec
INSERT INTO instances (
	id,
	deployment_id,
	workspace_id,
	project_id,
	region,
	k8s_name,
	address,
	cpu_millicores,
	memory_mib,
	status
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(deployment_id),
	sqlc.arg(workspace_id),
	sqlc.arg(project_id),
	sqlc.arg(region),
	sqlc.arg(k8s_name),
	sqlc.arg(address),
	sqlc.arg(cpu_millicores),
	sqlc.arg(memory_mib),
	sqlc.arg(status)
)
ON DUPLICATE KEY UPDATE
	address = sqlc.arg(address),
	cpu_millicores = sqlc.arg(cpu_millicores),
	memory_mib = sqlc.arg(memory_mib),
	status = sqlc.arg(status)
	;
