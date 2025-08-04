-- name: GetHealthyVMsByDeployment :many
SELECT id, deployment_id, metal_host_id, region, private_ip, port, cpu_millicores, memory_mb, status, health_status, last_heartbeat
FROM vms
WHERE deployment_id = ? 
  AND health_status = 'healthy' 
  AND last_heartbeat > ?
ORDER BY last_heartbeat DESC;

-- name: ClaimAvailableVM :exec
UPDATE vms
SET metal_host_id = ?, status = 'provisioning'
WHERE id = (
    SELECT id FROM (
        SELECT id FROM vms
        WHERE deployment_id = ? 
          AND region = ? 
          AND status = 'allocated'
        ORDER BY id
        LIMIT 1
    ) AS subquery
);

-- name: UpdateVMHeartbeat :exec
UPDATE vms
SET health_status = ?, last_heartbeat = ?
WHERE id = ?;

-- name: CreateAllocatedVM :exec
INSERT INTO vms (id, deployment_id, region, cpu_millicores, memory_mb, status, health_status)
VALUES (?, ?, ?, ?, ?, 'allocated', 'unknown');

-- name: ProvisionVM :exec
UPDATE vms
SET private_ip = ?, port = ?, status = 'starting'
WHERE id = ?;

-- name: UpdateVMStatus :exec
UPDATE vms
SET status = ?, health_status = ?, last_heartbeat = ?
WHERE id = ?;

-- name: GetVMsByHost :many
SELECT id, deployment_id, region, private_ip, port, status, health_status, last_heartbeat
FROM vms
WHERE metal_host_id = ?;

-- name: CountVMsByHost :one
SELECT COUNT(*) as count
FROM vms
WHERE metal_host_id = ? AND status IN ('running', 'starting');

-- name: DeleteVM :exec
DELETE FROM vms WHERE id = ?;

-- name: GetStaleVMs :many
SELECT id, deployment_id, metal_host_id, region, private_ip, port, last_heartbeat
FROM vms
WHERE last_heartbeat < ?
  AND status NOT IN ('stopped', 'allocated')
ORDER BY last_heartbeat;

-- name: GetAvailableVMSlots :many
SELECT id, deployment_id, region
FROM vms
WHERE deployment_id = ? 
  AND region = ? 
  AND status = 'allocated'
ORDER BY id
LIMIT ?;