-- name: FindDeploymentTopologyMinReplicas :many
-- Returns the per-region minimum replica requirement for a deployment.
-- Used by ReportDeploymentStatus to compute whether enough regions are
-- healthy to call DeployService.NotifyInstancesReady.
SELECT region_id, autoscaling_replicas_min
FROM deployment_topology
WHERE deployment_id = sqlc.arg(deployment_id);
