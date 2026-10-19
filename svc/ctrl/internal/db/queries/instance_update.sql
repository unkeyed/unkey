-- name: UpdateInstance :exec
UPDATE instances
SET
    address = sqlc.arg(address),
    cpu_millicores = sqlc.arg(cpu_millicores),
    memory_mib = sqlc.arg(memory_mib),
    status = sqlc.arg(status)
WHERE id = sqlc.arg(id);
