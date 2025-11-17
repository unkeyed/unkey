-- name: FindVMsByDeploymentId :many
SELECT id, deployment_id, metal_host_id, address, cpu_millicores, memory_mb, status 
FROM vms 
WHERE deployment_id = ?;