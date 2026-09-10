-- Replace the wildcard view introduced by the immutable app_id migration.
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
    network_ingress_public_bytes,
    network_ingress_private_bytes,
    attributes
FROM default.instance_checkpoints_v1
FINAL;
