-- name: FindSentinelDeployContextByK8sName :one
-- FindSentinelDeployContextByK8sName returns the sentinel's deploy status
-- along with its desired and observed running image. Used by
-- ReportSentinelStatus to determine whether to trigger NotifyReady — the
-- awakeable should only be resolved when the desired image is actually
-- running.
SELECT
    id,
    deploy_status,
    image AS desired_image,
    running_image
FROM sentinels
WHERE k8s_name = sqlc.arg(k8s_name) LIMIT 1;
