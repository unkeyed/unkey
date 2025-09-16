-- name: UpsertVM :exec
INSERT INTO vms (id, deployment_id, address, cpu_millicores, memory_mb, status)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  deployment_id = VALUES(deployment_id),
  address = VALUES(address),
  cpu_millicores = VALUES(cpu_millicores),
  memory_mb = VALUES(memory_mb),
  status = VALUES(status)
;
