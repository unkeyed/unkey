ALTER TABLE default.instance_checkpoints_v1
    ADD COLUMN IF NOT EXISTS network_egress_app_bytes Int64 DEFAULT 0 CODEC(Delta, ZSTD(1))
    AFTER network_egress_private_bytes;

CREATE OR REPLACE VIEW default.instance_checkpoints AS
SELECT
    workspace_id,
    project_id,
    app_id,
    environment_id,
    resource_type,
    resource_id,
    instance_id,
    container_uid,
    ts,
    event_kind,
    cpu_usage_usec,
    memory_bytes,
    cpu_allocated_millicores,
    memory_allocated_bytes,
    disk_allocated_bytes,
    disk_used_bytes,
    network_egress_public_bytes,
    network_egress_private_bytes,
    network_egress_app_bytes,
    network_ingress_public_bytes,
    network_ingress_private_bytes,
    attributes
FROM default.instance_checkpoints_v1
FINAL;
