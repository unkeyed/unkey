
-- name: InsertInstance :exec
INSERT INTO instances (
	id,
	deployment_id,
	workspace_id,
	project_id,
	region,
	shard,
	pod_name,
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
	sqlc.arg(shard),
	sqlc.arg(pod_name),
	sqlc.arg(address),
	sqlc.arg(cpu_millicores),
	sqlc.arg(memory_mib),
	sqlc.arg(status)
);
