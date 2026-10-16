-- name: UpdateInstance :exec
-- UpdateInstance refreshes an instance that is already known, matching on its
-- primary identity so only the target row is locked. A report for an existing
-- instance must not go through INSERT ... ON DUPLICATE KEY UPDATE: the insert
-- attempt probes the id and (k8s_name, region_id) unique indexes and allocates
-- an auto-increment value, and those gap locks deadlock under concurrent
-- reports.
UPDATE instances
SET
    address = sqlc.arg(address),
    cpu_millicores = sqlc.arg(cpu_millicores),
    memory_mib = sqlc.arg(memory_mib),
    status = sqlc.arg(status)
WHERE id = sqlc.arg(id);
