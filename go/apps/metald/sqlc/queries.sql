-- queries.sql

-- name: AllocateNetwork :one
UPDATE networks
SET is_allocated = 1
WHERE id = (
    SELECT id FROM networks
    WHERE is_allocated = 0
    ORDER BY id
    LIMIT 1
)
RETURNING *;

-- name: CreateNetworkAllocation :one
INSERT INTO network_allocations (deployment_id, network_id, available_ips, bridge_name)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetNetworkAllocation :one
SELECT na.*, n.base_network
FROM network_allocations na
JOIN networks n ON na.network_id = n.id
WHERE na.deployment_id = ?;

-- name: PopAvailableIPJSON :one
UPDATE network_allocations
SET available_ips = json_remove(available_ips, '$[0]')
WHERE deployment_id = ?
AND json_array_length(available_ips) > 0
RETURNING CAST(json_extract(available_ips, '$[0]') AS TEXT) AS ip, id;

-- name: AllocateIP :one
INSERT INTO ip_allocations (vm_id, ip_addr, network_allocation_id)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetIPAllocation :one
SELECT * FROM ip_allocations WHERE vm_id = ?;

-- name: ReleaseIP :exec
DELETE FROM ip_allocations WHERE vm_id = ?;

-- name: ReturnIPJSON :exec
UPDATE network_allocations
SET available_ips = json_insert(available_ips, '$[#]', ?)
WHERE deployment_id = ?;

-- name: GetAvailableIPCount :one
SELECT json_array_length(available_ips) as count
FROM network_allocations
WHERE deployment_id = ?;

-- name: ReleaseNetwork :exec
UPDATE networks
SET is_allocated = 0
WHERE id = ?;

-- name: DeleteIPAllocationsForNetwork :exec
DELETE FROM ip_allocations
WHERE network_allocation_id = ?;

-- name: DeleteNetworkAllocation :exec
DELETE FROM network_allocations
WHERE deployment_id = ?;

-- name: GetNetworkStats :one
SELECT
    (SELECT COUNT(*) FROM networks) as total_networks,
    (SELECT COUNT(*) FROM networks WHERE is_allocated = 0) as available_networks,
    (SELECT COUNT(*) FROM network_allocations) as active_deployments,
    (SELECT COUNT(*) FROM ip_allocations) as allocated_ips;
