
-- name: UpsertInstance :exec
INSERT INTO instances (
	id,
	deployment_id,
	workspace_id,
	project_id,
	region,
	address,
	cpu_millicores,
	memory_mb,
	status
)
VALUES (
	sqlc.arg(id),
	sqlc.arg(deployment_id),
	sqlc.arg(workspace_id),
	sqlc.arg(project_id),
	sqlc.arg(region),
	sqlc.arg(address),
	sqlc.arg(cpu_millicores),
	sqlc.arg(memory_mb),
	sqlc.arg(status)
)
ON DUPLICATE KEY UPDATE
	address = sqlc.arg(address),
	cpu_millicores = sqlc.arg(cpu_millicores),
	memory_mb = sqlc.arg(memory_mb),
	status = sqlc.arg(status)
;
