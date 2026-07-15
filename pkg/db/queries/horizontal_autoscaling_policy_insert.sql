-- name: InsertHorizontalAutoscalingPolicy :exec
INSERT INTO horizontal_autoscaling_policies (
    id,
    workspace_id,
    replicas_min,
    replicas_max,
    cpu_threshold,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(replicas_min),
    sqlc.arg(replicas_max),
    sqlc.arg(cpu_threshold),
    sqlc.arg(created_at)
);
