-- name: GetActiveHostsByRegion :many
SELECT id, region, availability_zone, instance_type, private_ip, capacity_cpu_millicores, capacity_memory_mb, allocated_cpu_millicores, allocated_memory_mb, last_heartbeat
FROM metal_hosts
WHERE region = ? 
  AND status = 'active'
  AND last_heartbeat > ?
ORDER BY allocated_cpu_millicores ASC;

-- name: UpdateHostHeartbeat :exec
UPDATE metal_hosts
SET allocated_cpu_millicores = ?, allocated_memory_mb = ?, last_heartbeat = ?
WHERE id = ?;

-- name: InsertMetalHost :exec
INSERT INTO metal_hosts (id, region, availability_zone, instance_type, ec2_instance_id, private_ip, status, capacity_cpu_millicores, capacity_memory_mb, allocated_cpu_millicores, allocated_memory_mb, last_heartbeat)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateHostStatus :exec
UPDATE metal_hosts
SET status = ?
WHERE id = ?;

-- name: GetHostCapacitySummary :many
SELECT region, availability_zone, COUNT(*) as host_count, 
       SUM(capacity_cpu_millicores) as total_cpu, 
       SUM(allocated_cpu_millicores) as allocated_cpu,
       SUM(capacity_memory_mb) as total_memory,
       SUM(allocated_memory_mb) as allocated_memory
FROM metal_hosts
WHERE status = 'active'
GROUP BY region, availability_zone;

-- name: GetStaleHosts :many
SELECT id, region, ec2_instance_id, last_heartbeat
FROM metal_hosts
WHERE last_heartbeat < ?
  AND status != 'terminated'
ORDER BY last_heartbeat;