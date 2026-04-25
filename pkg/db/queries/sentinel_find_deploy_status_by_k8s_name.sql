-- name: FindSentinelDeployContextByK8sName :one
-- Returns the sentinel fields ReportSentinelStatus needs to decide whether
-- a rollout has converged: deploy_status (gates), image comparison, and
-- desired replica count.
SELECT
    id,
    deploy_status,
    image AS desired_image,
    running_image,
    desired_replicas
FROM sentinels
WHERE k8s_name = sqlc.arg(k8s_name) LIMIT 1;
