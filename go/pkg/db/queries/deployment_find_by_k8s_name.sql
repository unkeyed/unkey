-- name: FindDeploymentByK8sName :one
SELECT * FROM `deployments` WHERE k8s_name = sqlc.arg(k8s_name);
