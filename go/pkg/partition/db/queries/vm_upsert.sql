-- name: UpsertVM :exec
INSERT INTO vms (id, deployment_id, region, private_ip, port, cpu_millicores, memory_mb, status, health_status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  deployment_id = VALUES(deployment_id),
  region = VALUES(region),
  private_ip = VALUES(private_ip),
  port = VALUES(port),
  cpu_millicores = VALUES(cpu_millicores),
  memory_mb = VALUES(memory_mb),
  status = VALUES(status),
  health_status = VALUES(health_status);
